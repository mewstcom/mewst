package seed

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// testRosterPasswordは、テスト用の名簿が共有するパスワード。ユーザー行へ
// 届いたダイジェストと突き合わせる。すべてのアカウントが作成される理由は、それで
// サインインできることであるため。
const testRosterPassword = "seed-password"

// testAccountSeqは、あるテストのアカウントのatnameとアドレスを、別の
// テストのものと分ける。このパッケージのテストはいずれもロールバックされる自身の
// トランザクションで実行されるが、同時に走る2つは、profiles.atnameとusers.email
// の一意インデックスの上で出会うことになる。
var testAccountSeq atomic.Int64

// newTestRosterは、役割ごとに1件のアカウントを持つ名簿を、この呼び出しに
// 固有のatnameとアドレスで組み立てる。
func newTestRoster(t *testing.T) *userRoster {
	t.Helper()

	digest, err := auth.HashPassword(testRosterPassword)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}

	users := make([]rosterUser, 0, len(allSeedRoles))
	for _, role := range allSeedRoles {
		users = append(users, newTestRosterUser(t, role))
	}

	return &userRoster{
		path:           "seed-users.toml",
		passwordDigest: digest,
		users:          users,
	}
}

// newTestRosterUserは1件を組み立て、名簿がその役割へ与えることになっている
// ロケール・タイムゾーン・フラグを与える。
func newTestRosterUser(t *testing.T, role seedRole) rosterUser {
	t.Helper()

	// atnameはURLに入るため、カラムに収まるかどうかではなく、すべての
	// アカウントのatnameが満たす規則に合わせて組み立てる。
	atname := fmt.Sprintf("sd%d", testAccountSeq.Add(1)+time.Now().UnixNano()%1_000_000)
	if !validator.IsValidAtname(atname) {
		t.Fatalf("テスト用のatname %qがアカウントの規則を満たしていない", atname)
	}

	entry := rosterUser{
		role:     role,
		atname:   atname,
		name:     fmt.Sprintf("%s のアカウント", role),
		email:    atname + "@example.com",
		note:     fmt.Sprintf("%s の確認用", role),
		locale:   "ja",
		timeZone: "Asia/Tokyo",
	}

	switch role {
	case roleEnglish:
		entry.locale = "en"
		entry.timeZone = "Etc/UTC"
		entry.featureFlags = slices.Clone(model.AllFeatureFlagNames)
	case roleMain, roleNewcomer:
		entry.featureFlags = slices.Clone(model.AllFeatureFlagNames)
	case roleFollower, roleDiscarded:
		entry.featureFlags = nil
	}

	return entry
}

// TestCreateAccountsは、名簿が挙げるすべてのアカウントが、アカウントを構成する
// 4つの行として作成され、名簿がそのアカウントについて述べたことと、その役割が示す
// ためにあるものを持つことを検証する。
func TestCreateAccounts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	roster := newTestRoster(t)

	// 時点はカラムが保持できる精度へ切り詰める。timestamp(6) は与えられた値を
	// 丸めるため、ナノ秒を持つ時点は、書き込んだ値からマイクロ秒未満だけずれた値と
	// して返ってくる。
	now := time.Now().Truncate(time.Microsecond)

	accounts, err := createAccounts(ctx, tx, roster, now)
	if err != nil {
		t.Fatalf("アカウントの作成に失敗: %v", err)
	}

	// アカウントは名簿が挙げる順で返る。実行がそれらを報告する順序であるため。
	if len(accounts) != len(roster.users) {
		t.Fatalf("作成されたアカウント数 = %d、期待値 = %d", len(accounts), len(roster.users))
	}
	for i, account := range accounts {
		if account.roster.role != roster.users[i].role {
			t.Errorf("%d件目の役割 = %s、期待値 = %s", i+1, account.roster.role, roster.users[i].role)
		}
	}

	byRole := make(map[seedRole]seedAccount, len(accounts))
	for _, account := range accounts {
		byRole[account.roster.role] = account
	}

	tests := []struct {
		role             seedRole
		wantLocale       string
		wantTimeZone     string
		wantFeatureFlags []model.FeatureFlagName
		wantDiscarded    bool
	}{
		{
			role:             roleMain,
			wantLocale:       "ja",
			wantTimeZone:     "Asia/Tokyo",
			wantFeatureFlags: model.AllFeatureFlagNames,
		},
		{
			// followerがフラグを1つも持たないのは意図的である。フラグの
			// 内側にある画面を見比べる相手であるため。
			role:         roleFollower,
			wantLocale:   "ja",
			wantTimeZone: "Asia/Tokyo",
		},
		{
			role:             roleEnglish,
			wantLocale:       "en",
			wantTimeZone:     "Etc/UTC",
			wantFeatureFlags: model.AllFeatureFlagNames,
		},
		{
			role:             roleNewcomer,
			wantLocale:       "ja",
			wantTimeZone:     "Asia/Tokyo",
			wantFeatureFlags: model.AllFeatureFlagNames,
		},
		{
			role:          roleDiscarded,
			wantLocale:    "ja",
			wantTimeZone:  "Asia/Tokyo",
			wantDiscarded: true,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			account, ok := byRole[tt.role]
			if !ok {
				t.Fatalf("役割%sのアカウントが作成されていない", tt.role)
			}

			assertUserRow(t, ctx, tx, account, tt.wantLocale, tt.wantTimeZone)
			assertProfileRow(t, ctx, tx, account, tt.wantDiscarded)
			assertLinkedRows(t, ctx, tx, account)
			assertFeatureFlags(t, ctx, tx, account, tt.wantFeatureFlags)
		})
	}

	// 3年分のポストを持つことになるアカウントは、その最も古いものより前に
	// 参加している必要があり、newcomerは今まさに参加している必要がある。この役割が
	// あるのは、サインアップ直後に見える画面のためであるため。
	if joined := byRole[roleMain].profile.JoinedAt; joined.After(now.AddDate(0, -historyMonths, 0)) {
		t.Errorf("mainのjoined_at = %v、期待値 = %dか月以上前", joined, historyMonths)
	}
	if joined := byRole[roleNewcomer].profile.JoinedAt; joined.Before(now.AddDate(0, 0, -1)) {
		t.Errorf("newcomerのjoined_at = %v、期待値 = 実行時点", joined)
	}
}

// assertUserRowは、サインインの照合先となる行を確認する。
func assertUserRow(t *testing.T, ctx context.Context, tx *sql.Tx, account seedAccount, wantLocale, wantTimeZone string) {
	t.Helper()

	var (
		email          string
		passwordDigest string
		locale         string
		timeZone       string
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT email, password_digest, locale, time_zone FROM users WHERE id = $1
	`, uuid.UUID(account.user.ID)).Scan(&email, &passwordDigest, &locale, &timeZone); err != nil {
		t.Fatalf("ユーザー行の取得に失敗: %v", err)
	}

	if email != account.roster.email {
		t.Errorf("email = %q、期待値 = %q", email, account.roster.email)
	}

	// シードが作ったアカウントの意義はそれでサインインできることであり、共通
	// パスワードは平文ではなくダイジェストとして行へ届く。
	if err := auth.CheckPassword(passwordDigest, testRosterPassword); err != nil {
		t.Errorf("名簿のパスワードで照合できない: %v", err)
	}

	// ロケールとタイムゾーンは、エクスポートのHTMLが文言・日時表記・月境界を
	// 出し分ける元であり、名簿がアカウントごとにこれを持つのはそのためである。
	if locale != wantLocale {
		t.Errorf("locale = %q、期待値 = %q", locale, wantLocale)
	}
	if timeZone != wantTimeZone {
		t.Errorf("time_zone = %q、期待値 = %q", timeZone, wantTimeZone)
	}
}

// assertProfileRowは、すべての画面がアカウントを表示する元になる行を確認する。
func assertProfileRow(t *testing.T, ctx context.Context, tx *sql.Tx, account seedAccount, wantDiscarded bool) {
	t.Helper()

	var (
		ownerType   string
		atname      string
		name        string
		description string
		avatarKind  string
		discardedAt sql.NullTime
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT owner_type, atname, name, description, avatar_kind, discarded_at
		FROM profiles WHERE id = $1
	`, uuid.UUID(account.profile.ID)).Scan(&ownerType, &atname, &name, &description, &avatarKind, &discardedAt); err != nil {
		t.Fatalf("プロフィール行の取得に失敗: %v", err)
	}

	if ownerType != model.ProfileOwnerTypeUser {
		t.Errorf("owner_type = %q、期待値 = %q", ownerType, model.ProfileOwnerTypeUser)
	}
	if atname != account.roster.atname {
		t.Errorf("atname = %q、期待値 = %q", atname, account.roster.atname)
	}
	if name != account.roster.name {
		t.Errorf("name = %q、期待値 = %q", name, account.roster.name)
	}
	if avatarKind == "" {
		t.Error("avatar_kindが空。アカウント作成が与えるのと同じ既定値が入っていない")
	}

	// 何も書かれていないプロフィールは、本番のどのアカウントも取らない状態で
	// あり、自己紹介を表示する画面は空の状態でしか見られないことになる。
	if description == "" {
		t.Error("descriptionが空。役割ごとの自己紹介が入っていない")
	}

	// プロフィールが削除済みなのはdiscarded役割だけである。作者が居なくなった
	// ポストが他の人にどう見えるのかを示すのがこの役割であるため。
	if discardedAt.Valid != wantDiscarded {
		t.Errorf("discarded_atが設定されている = %t、期待値 = %t", discardedAt.Valid, wantDiscarded)
	}

	// モデルは後続の生成器へ渡されるため、行が述べているのと同じことを述べて
	// いる必要がある。
	if (account.profile.DiscardedAt != nil) != wantDiscarded {
		t.Errorf("返されたプロフィールのDiscardedAt = %v、設定されている = %tを期待", account.profile.DiscardedAt, wantDiscarded)
	}
}

// assertLinkedRowsは、ユーザーとプロフィールを結ぶ2つの行を確認する。
func assertLinkedRows(t *testing.T, ctx context.Context, tx *sql.Tx, account seedAccount) {
	t.Helper()

	if account.actor.UserID != account.user.ID || account.actor.ProfileID != account.profile.ID {
		t.Errorf(
			"アクターが結んでいるのはuser %v / profile %vで、アカウントのuser %v / profile %vではない",
			account.actor.UserID, account.actor.ProfileID, account.user.ID, account.profile.ID,
		)
	}

	for _, table := range []string{"user_profiles", "actors"} {
		var count int
		if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT COUNT(*) FROM %s WHERE user_id = $1 AND profile_id = $2
		`, table), uuid.UUID(account.user.ID), uuid.UUID(account.profile.ID)).Scan(&count); err != nil {
			t.Fatalf("%sの件数の取得に失敗: %v", table, err)
		}
		if count != 1 {
			t.Errorf("%sの件数 = %d、期待値 = 1", table, count)
		}
	}
}

// assertFeatureFlagsは、アカウントへ付与されたフラグを確認する。
func assertFeatureFlags(t *testing.T, ctx context.Context, tx *sql.Tx, account seedAccount, want []model.FeatureFlagName) {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT name FROM feature_flags WHERE actor_id = $1 ORDER BY name
	`, uuid.UUID(account.actor.ID))
	if err != nil {
		t.Fatalf("フィーチャーフラグの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var got []model.FeatureFlagName
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("フィーチャーフラグ名の読み取りに失敗: %v", err)
		}
		got = append(got, model.FeatureFlagName(name))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("フィーチャーフラグの読み取りに失敗: %v", err)
	}

	wantSorted := slices.Clone(want)
	slices.Sort(wantSorted)
	if !slices.Equal(got, wantSorted) {
		t.Errorf("付与されたフィーチャーフラグ = %v、期待値 = %v", got, wantSorted)
	}
}

// TestCreateAccounts_ReportsTheRoleThatFailedは、失敗がその下の行ではなく
// 役割を名指しすることを検証する。
//
// 名簿の該当箇所を見つけるのに使うのも、生成器がそのアカウントを指すのに使うのも役割で
// あり、どの記載を見ればよいのかを開発者が読み取るために要るのはそれであるため。
func TestCreateAccounts_ReportsTheRoleThatFailed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	roster := newTestRoster(t)

	// atnameを共有する2件は、実行がデータベースへ辿り着く前に名簿自身の
	// 検査が拒否するもの。ここでは、名簿の途中で失敗する書き込み全般の代わりに置いて
	// いる。
	roster.users[1].atname = roster.users[0].atname

	_, err := createAccounts(ctx, tx, roster, time.Now())
	if err == nil {
		t.Fatal("createAccounts() がエラーを返さなかった")
	}

	want := fmt.Sprintf("役割%s", roster.users[1].role)
	if got := err.Error(); !strings.Contains(got, want) {
		t.Errorf("エラーが%qを含むことを期待したが%qだった", want, got)
	}
}

// TestRoleProfilesCoverEveryRoleは、生成器が名指しするすべての役割に、その
// プロフィールが記述されていることを検証する。
//
// allSeedRolesへ追加されたのにroleProfilesから漏れた役割は、失敗しない。その
// アカウントは、今日参加し、自身について何も書かれていない状態で作成されるだけで
// ある。それは、自己紹介が画面をそこから遠ざけるために存在している、まさにその状態で
// ある。
func TestRoleProfilesCoverEveryRole(t *testing.T) {
	t.Parallel()

	for _, role := range allSeedRoles {
		shape, ok := roleProfiles[role]
		if !ok {
			t.Errorf("役割%sのroleProfilesが無い", role)

			continue
		}
		if shape.description == "" {
			t.Errorf("役割%sのdescriptionが空", role)
		}
	}

	for role := range roleProfiles {
		if !slices.Contains(allSeedRoles, role) {
			t.Errorf("roleProfilesの%sは生成器が名指しする役割ではない", role)
		}
	}
}

// TestAccountForRoleは、役割を求めた生成器がそのために作成されたアカウントを
// 受け取ること、そしてそれが無い場合はどの役割が欠けているのかを告げられることを
// 検証する。
func TestAccountForRole(t *testing.T) {
	t.Parallel()

	accounts := []seedAccount{
		{roster: rosterUser{role: roleMain}},
		{roster: rosterUser{role: roleFollower}},
	}

	for _, role := range []seedRole{roleMain, roleFollower} {
		account, err := accountForRole(accounts, role)
		if err != nil {
			t.Fatalf("役割%sのアカウントの取得に失敗: %v", role, err)
		}
		if account.roster.role != role {
			t.Errorf("役割 = %s、期待値 = %s", account.roster.role, role)
		}
	}

	// 失敗はその役割を名指しする。いくつも先の文で、存在しないプロフィールと
	// して現れるのではなく。
	_, err := accountForRole(accounts, roleNewcomer)
	if err == nil {
		t.Fatal("作成されていない役割でエラーが返らなかった")
	}
	if !strings.Contains(err.Error(), string(roleNewcomer)) {
		t.Errorf("エラー = %q、役割%sを含むことを期待", err, roleNewcomer)
	}
}
