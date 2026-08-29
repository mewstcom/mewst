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

// testRosterPassword is the shared password the test roster is built around.
// It is checked against the digest that reaches the user row, because the one
// thing every account is created for is being able to sign in as it.
//
// [Ja] testRosterPassword は、テスト用の名簿が共有するパスワード。ユーザー行へ
// 届いたダイジェストと突き合わせる。すべてのアカウントが作成される理由は、それで
// サインインできることであるため。
const testRosterPassword = "seed-password"

// testAccountSeq keeps the atnames and addresses of one test's accounts apart
// from another's. Every test in this package runs in a transaction of its own
// that is rolled back, but two of them running at once would still meet on the
// unique indexes over profiles.atname and users.email.
//
// [Ja] testAccountSeq は、あるテストのアカウントの atname とアドレスを、別の
// テストのものと分ける。このパッケージのテストはいずれもロールバックされる自身の
// トランザクションで実行されるが、同時に走る 2 つは、profiles.atname と users.email
// の一意インデックスの上で出会うことになる。
var testAccountSeq atomic.Int64

// newTestRoster builds a roster holding one account per role, with atnames and
// addresses unique to this call.
//
// [Ja] newTestRoster は、役割ごとに 1 件のアカウントを持つ名簿を、この呼び出しに
// 固有の atname とアドレスで組み立てる。
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

// newTestRosterUser builds one entry, giving the role the locale, the time
// zone and the flags the roster is meant to give it.
//
// [Ja] newTestRosterUser は 1 件を組み立て、名簿がその役割へ与えることになっている
// ロケール・タイムゾーン・フラグを与える。
func newTestRosterUser(t *testing.T, role seedRole) rosterUser {
	t.Helper()

	// The atname goes into a URL, so it is built to the rule every account's
	// atname has to satisfy rather than to whatever fits in the column.
	//
	// [Ja] atname は URL に入るため、カラムに収まるかどうかではなく、すべての
	// アカウントの atname が満たす規則に合わせて組み立てる。
	atname := fmt.Sprintf("sd%d", testAccountSeq.Add(1)+time.Now().UnixNano()%1_000_000)
	if !validator.IsValidAtname(atname) {
		t.Fatalf("テスト用の atname %q がアカウントの規則を満たしていない", atname)
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

// TestCreateAccounts verifies that every account the roster names is created
// as the four rows an account is made of, holding what the roster said about
// it and what its role is there to show.
//
// [Ja] TestCreateAccounts は、名簿が挙げるすべてのアカウントが、アカウントを構成する
// 4 つの行として作成され、名簿がそのアカウントについて述べたことと、その役割が示す
// ためにあるものを持つことを検証する。
func TestCreateAccounts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	roster := newTestRoster(t)

	// The instant is truncated to what the column can hold. A timestamp(6)
	// rounds what it is given, so an instant carrying nanoseconds would come
	// back a fraction of a microsecond on either side of what was written.
	//
	// [Ja] 時点はカラムが保持できる精度へ切り詰める。timestamp(6) は与えられた値を
	// 丸めるため、ナノ秒を持つ時点は、書き込んだ値からマイクロ秒未満だけずれた値と
	// して返ってくる。
	now := time.Now().Truncate(time.Microsecond)

	accounts, err := createAccounts(ctx, tx, roster, now)
	if err != nil {
		t.Fatalf("アカウントの作成に失敗: %v", err)
	}

	// The accounts come back in the order the roster names them, which is the
	// order a run reports them in.
	//
	// [Ja] アカウントは名簿が挙げる順で返る。実行がそれらを報告する順序であるため。
	if len(accounts) != len(roster.users) {
		t.Fatalf("作成されたアカウント数 = %d, want %d", len(accounts), len(roster.users))
	}
	for i, account := range accounts {
		if account.roster.role != roster.users[i].role {
			t.Errorf("%d 件目の役割 = %s, want %s", i+1, account.roster.role, roster.users[i].role)
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
			// The follower holds no flag on purpose: it is what a screen
			// behind a flag is compared against.
			//
			// [Ja] follower がフラグを 1 つも持たないのは意図的である。フラグの
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
				t.Fatalf("役割 %s のアカウントが作成されていない", tt.role)
			}

			assertUserRow(t, ctx, tx, account, tt.wantLocale, tt.wantTimeZone)
			assertProfileRow(t, ctx, tx, account, tt.wantDiscarded)
			assertLinkedRows(t, ctx, tx, account)
			assertFeatureFlags(t, ctx, tx, account, tt.wantFeatureFlags)
		})
	}

	// An account that is to hold three years of posts has to have joined
	// before the oldest of them, and the newcomer has to have joined just now:
	// the screens it is there for are the ones seen right after signing up.
	//
	// [Ja] 3 年分のポストを持つことになるアカウントは、その最も古いものより前に
	// 参加している必要があり、newcomer は今まさに参加している必要がある。この役割が
	// あるのは、サインアップ直後に見える画面のためであるため。
	if joined := byRole[roleMain].profile.JoinedAt; joined.After(now.AddDate(0, -historyMonths, 0)) {
		t.Errorf("main の joined_at = %v, want %d か月以上前", joined, historyMonths)
	}
	if joined := byRole[roleNewcomer].profile.JoinedAt; joined.Before(now.AddDate(0, 0, -1)) {
		t.Errorf("newcomer の joined_at = %v, want 実行時点", joined)
	}
}

// assertUserRow checks the row a sign-in is made against.
//
// [Ja] assertUserRow は、サインインの照合先となる行を確認する。
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
		t.Errorf("email = %q, want %q", email, account.roster.email)
	}

	// The whole point of an account the seed created is signing in as it, and
	// the shared password reaches the row as a digest rather than as itself.
	//
	// [Ja] シードが作ったアカウントの意義はそれでサインインできることであり、共通
	// パスワードは平文ではなくダイジェストとして行へ届く。
	if err := auth.CheckPassword(passwordDigest, testRosterPassword); err != nil {
		t.Errorf("名簿のパスワードで照合できない: %v", err)
	}

	// The locale and the time zone are what the export HTML picks its wording,
	// its date format and its month boundaries from, which is why the roster
	// carries them per account.
	//
	// [Ja] ロケールとタイムゾーンは、エクスポートの HTML が文言・日時表記・月境界を
	// 出し分ける元であり、名簿がアカウントごとにこれを持つのはそのためである。
	if locale != wantLocale {
		t.Errorf("locale = %q, want %q", locale, wantLocale)
	}
	if timeZone != wantTimeZone {
		t.Errorf("time_zone = %q, want %q", timeZone, wantTimeZone)
	}
}

// assertProfileRow checks the row every screen shows the account through.
//
// [Ja] assertProfileRow は、すべての画面がアカウントを表示する元になる行を確認する。
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
		t.Errorf("owner_type = %q, want %q", ownerType, model.ProfileOwnerTypeUser)
	}
	if atname != account.roster.atname {
		t.Errorf("atname = %q, want %q", atname, account.roster.atname)
	}
	if name != account.roster.name {
		t.Errorf("name = %q, want %q", name, account.roster.name)
	}
	if avatarKind == "" {
		t.Error("avatar_kind が空。アカウント作成が与えるのと同じ既定値が入っていない")
	}

	// A profile with nothing written about it is a state no account in
	// production is in, and the screens that show a self-introduction would
	// only ever be looked at empty.
	//
	// [Ja] 何も書かれていないプロフィールは、本番のどのアカウントも取らない状態で
	// あり、自己紹介を表示する画面は空の状態でしか見られないことになる。
	if description == "" {
		t.Error("description が空。役割ごとの自己紹介が入っていない")
	}

	// The discarded role is the only one whose profile is deleted: it is what
	// shows how a post whose author is gone appears to everyone else.
	//
	// [Ja] プロフィールが削除済みなのは discarded 役割だけである。作者が居なくなった
	// ポストが他の人にどう見えるのかを示すのがこの役割であるため。
	if discardedAt.Valid != wantDiscarded {
		t.Errorf("discarded_at が設定されている = %t, want %t", discardedAt.Valid, wantDiscarded)
	}

	// The model is handed on to the generators that follow, so it has to say
	// the same thing the row does.
	//
	// [Ja] モデルは後続の生成器へ渡されるため、行が述べているのと同じことを述べて
	// いる必要がある。
	if (account.profile.DiscardedAt != nil) != wantDiscarded {
		t.Errorf("返されたプロフィールの DiscardedAt = %v, want 設定されている = %t", account.profile.DiscardedAt, wantDiscarded)
	}
}

// assertLinkedRows checks the two rows that tie a user and a profile together.
//
// [Ja] assertLinkedRows は、ユーザーとプロフィールを結ぶ 2 つの行を確認する。
func assertLinkedRows(t *testing.T, ctx context.Context, tx *sql.Tx, account seedAccount) {
	t.Helper()

	if account.actor.UserID != account.user.ID || account.actor.ProfileID != account.profile.ID {
		t.Errorf(
			"アクターが結んでいるのは user %v / profile %v で、アカウントの user %v / profile %v ではない",
			account.actor.UserID, account.actor.ProfileID, account.user.ID, account.profile.ID,
		)
	}

	for _, table := range []string{"user_profiles", "actors"} {
		var count int
		if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT COUNT(*) FROM %s WHERE user_id = $1 AND profile_id = $2
		`, table), uuid.UUID(account.user.ID), uuid.UUID(account.profile.ID)).Scan(&count); err != nil {
			t.Fatalf("%s の件数の取得に失敗: %v", table, err)
		}
		if count != 1 {
			t.Errorf("%s の件数 = %d, want 1", table, count)
		}
	}
}

// assertFeatureFlags checks the flags the account was granted.
//
// [Ja] assertFeatureFlags は、アカウントへ付与されたフラグを確認する。
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
		t.Errorf("付与されたフィーチャーフラグ = %v, want %v", got, wantSorted)
	}
}

// TestCreateAccounts_ReportsTheRoleThatFailed verifies that a failure names the
// role rather than the row underneath it.
//
// The role is what the roster entry is found by and what the generators refer
// to the account as, so it is what a developer needs in order to see which
// entry to look at.
//
// [Ja] TestCreateAccounts_ReportsTheRoleThatFailed は、失敗がその下の行ではなく
// 役割を名指しすることを検証する。
//
// 名簿の該当箇所を見つけるのに使うのも、生成器がそのアカウントを指すのに使うのも役割で
// あり、どの記載を見ればよいのかを開発者が読み取るために要るのはそれであるため。
func TestCreateAccounts_ReportsTheRoleThatFailed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	roster := newTestRoster(t)

	// Two accounts sharing an atname is what the roster's own checks reject
	// before a run reaches the database. Here it stands in for any write that
	// fails partway through the roster.
	//
	// [Ja] atname を共有する 2 件は、実行がデータベースへ辿り着く前に名簿自身の
	// 検査が拒否するもの。ここでは、名簿の途中で失敗する書き込み全般の代わりに置いて
	// いる。
	roster.users[1].atname = roster.users[0].atname

	_, err := createAccounts(ctx, tx, roster, time.Now())
	if err == nil {
		t.Fatal("createAccounts() がエラーを返さなかった")
	}

	want := fmt.Sprintf("役割 %s", roster.users[1].role)
	if got := err.Error(); !strings.Contains(got, want) {
		t.Errorf("エラーが %q を含むことを期待したが %q だった", want, got)
	}
}

// TestRoleProfilesCoverEveryRole verifies that every role a generator names has
// a profile described for it.
//
// A role added to allSeedRoles but left out of roleProfiles would not fail:
// the account would simply be created having joined today with nothing written
// about it, which is the state the descriptions exist to keep the screens out
// of.
//
// [Ja] TestRoleProfilesCoverEveryRole は、生成器が名指しするすべての役割に、その
// プロフィールが記述されていることを検証する。
//
// allSeedRoles へ追加されたのに roleProfiles から漏れた役割は、失敗しない。その
// アカウントは、今日参加し、自身について何も書かれていない状態で作成されるだけで
// ある。それは、自己紹介が画面をそこから遠ざけるために存在している、まさにその状態で
// ある。
func TestRoleProfilesCoverEveryRole(t *testing.T) {
	t.Parallel()

	for _, role := range allSeedRoles {
		shape, ok := roleProfiles[role]
		if !ok {
			t.Errorf("役割 %s の roleProfiles が無い", role)

			continue
		}
		if shape.description == "" {
			t.Errorf("役割 %s の description が空", role)
		}
	}

	for role := range roleProfiles {
		if !slices.Contains(allSeedRoles, role) {
			t.Errorf("roleProfiles の %s は生成器が名指しする役割ではない", role)
		}
	}
}
