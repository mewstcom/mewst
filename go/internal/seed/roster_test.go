package seed

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// validRosterはすべての検査を通る名簿。以下のテストは、これを1箇所ずつ
// 壊して確認する。
const validRoster = `
password = "seed-password"

[[users]]
role = "main"
atname = "seeduser1"
name = "シードユーザー 1"
email = "seeduser1@example.com"
note = "主な確認対象"
locale = "ja"
time_zone = "Asia/Tokyo"
feature_flags = "all"

[[users]]
role = "follower"
atname = "seeduser2"
name = "シードユーザー 2"
email = "seeduser2@example.com"
note = "main と相互フォロー"
locale = "ja"
time_zone = "Asia/Tokyo"
feature_flags = []

[[users]]
role = "english"
atname = "seeduser3"
name = "シードユーザー 3"
email = "seeduser3@example.com"
note = "en の出し分けを確認する"
locale = "en"
time_zone = "Etc/UTC"
feature_flags = "all"

[[users]]
role = "newcomer"
atname = "seeduser4"
name = "シードユーザー 4"
email = "seeduser4@example.com"
note = "ポスト 0 件"
locale = "ja"
time_zone = "Asia/Tokyo"
feature_flags = "all"

[[users]]
role = "discarded"
atname = "seeduser5"
name = "シードユーザー 5"
email = "seeduser5@example.com"
note = "削除済みプロフィール"
locale = "ja"
time_zone = "Asia/Tokyo"
feature_flags = []
`

func TestLoadUserRoster(t *testing.T) {
	t.Parallel()

	path := writeRoster(t, validRoster)

	roster, err := loadUserRoster(path)
	if err != nil {
		t.Fatalf("名簿の読み込みに失敗: %v", err)
	}

	// パスを名簿と一緒に持つのは、実行がどのファイルを読んだのかを報告する
	// ため。
	if roster.path != path {
		t.Errorf("名簿のパスが%qであることを期待したが%qだった", path, roster.path)
	}
	// 実行が書き込むのはダイジェストであるため、ファイルに書いたパスワードで
	// それを検証することが、ファイルのパスワードが読めていることの確認になる。
	if err := auth.CheckPassword(roster.passwordDigest, "seed-password"); err != nil {
		t.Errorf("パスワードダイジェストが名簿のパスワードと一致しない: %v", err)
	}
	if len(roster.users) != len(allSeedRoles) {
		t.Fatalf("アカウントが%d件であることを期待したが%d件だった", len(allSeedRoles), len(roster.users))
	}

	mainUser := roster.users[0]
	if mainUser.role != roleMain {
		t.Errorf("1件目の役割が%sであることを期待したが%sだった", roleMain, mainUser.role)
	}
	if mainUser.atname != "seeduser1" || mainUser.name != "シードユーザー 1" || mainUser.email != "seeduser1@example.com" {
		t.Errorf("1件目の内容がファイルと一致しない: %+v", mainUser)
	}
	if mainUser.note != "主な確認対象" {
		t.Errorf("1件目の覚え書きが%qであることを期待したが%qだった", "主な確認対象", mainUser.note)
	}
	if mainUser.locale != "ja" || mainUser.timeZone != "Asia/Tokyo" {
		t.Errorf("1件目のロケールとタイムゾーンがja / Asia/Tokyoであることを期待したが%q / %qだった", mainUser.locale, mainUser.timeZone)
	}
	if !slices.Equal(mainUser.featureFlags, model.AllFeatureFlagNames) {
		t.Errorf("feature_flagsが \"all\" のとき全フラグを期待したが%vだった", mainUser.featureFlags)
	}

	follower := roster.users[1]
	if len(follower.featureFlags) != 0 {
		t.Errorf("feature_flagsが空配列のときフラグ無しを期待したが%vだった", follower.featureFlags)
	}

	english := roster.users[2]
	if english.locale != "en" || english.timeZone != "Etc/UTC" {
		t.Errorf("3件目のロケールとタイムゾーンがen / Etc/UTCであることを期待したが%q / %qだった", english.locale, english.timeZone)
	}

	// アカウントはファイルが書いた順のまま保持する。上の各件を位置で読み取れる
	// のはそのためであり、この順序は実行がアカウントを作成する順と報告する順も
	// 決める。
	discarded := roster.users[4]
	if discarded.role != roleDiscarded {
		t.Errorf("5件目の役割が%sであることを期待したが%sだった", roleDiscarded, discarded.role)
	}
}

// TestLoadUserRosterAcceptsSelectedFeatureFlagsはfeature_flagsが取りうる
// 3つ目の形を確認する。全件でも0件でもなく、そのアカウントが持つと名指しされた
// フラグである。
func TestLoadUserRosterAcceptsSelectedFeatureFlags(t *testing.T) {
	t.Parallel()

	body := strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = ["go_example"]`, 1)

	roster, err := loadUserRoster(writeRoster(t, body))
	if err != nil {
		t.Fatalf("名簿の読み込みに失敗: %v", err)
	}

	want := []model.FeatureFlagName{model.FeatureFlagExample}
	if !slices.Equal(roster.users[0].featureFlags, want) {
		t.Errorf("フィーチャーフラグが%vであることを期待したが%vだった", want, roster.users[0].featureFlags)
	}
}

// TestLoadUserRosterTrimsNameAndNoteは、名前と覚え書きの前後の空白がアカウント
// へ持ち込まれないことを確認する。これらは自身の形式を持たない必須文字列であるため、
// 紛れ込んだ空白を他の検査が捕まえることはなく、そのアカウントが現れるすべての画面と、
// 実行の最後の報告に出てしまう。
func TestLoadUserRosterTrimsNameAndNote(t *testing.T) {
	t.Parallel()

	body := strings.Replace(validRoster, `name = "シードユーザー 1"`, `name = "  シードユーザー 1  "`, 1)
	body = strings.Replace(body, `note = "主な確認対象"`, `note = "  主な確認対象  "`, 1)

	roster, err := loadUserRoster(writeRoster(t, body))
	if err != nil {
		t.Fatalf("名簿の読み込みに失敗: %v", err)
	}

	if got := roster.users[0].name; got != "シードユーザー 1" {
		t.Errorf("表示名が%qであることを期待したが%qだった", "シードユーザー 1", got)
	}
	if got := roster.users[0].note; got != "主な確認対象" {
		t.Errorf("覚え書きが%qであることを期待したが%qだった", "主な確認対象", got)
	}
}

func TestLoadUserRosterRejectsMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed-users.toml")

	_, err := loadUserRoster(path)
	if err == nil {
		t.Fatal("名簿が無いときのエラーを期待したがnilだった")
	}

	// メッセージが見本を名指しするのは、それをコピーすることがこの状態の
	// 直し方であるため。
	if !strings.Contains(err.Error(), rosterExamplePath) {
		t.Errorf("エラーが%qを案内することを期待したが%qだった", rosterExamplePath, err)
	}
}

func TestLoadUserRosterRejectsInvalidRoster(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "TOMLとして読めないとき",
			body: "password = ",
			want: "読み込みに失敗",
		},
		{
			name: "知らないキーがあるとき",
			body: strings.Replace(validRoster, `note = "主な確認対象"`, `notes = "主な確認対象"`, 1),
			want: "知らないキー",
		},
		{
			name: "パスワードが空のとき",
			body: strings.Replace(validRoster, `password = "seed-password"`, `password = ""`, 1),
			want: "passwordが空です",
		},
		{
			name: "パスワードにLFがあるとき",
			body: strings.Replace(validRoster, `password = "seed-password"`, `password = "line1\nline2"`, 1),
			want: "passwordにCR / LFは含められません",
		},
		{
			name: "パスワードにCRがあるとき",
			body: strings.Replace(validRoster, `password = "seed-password"`, `password = "line1\rline2"`, 1),
			want: "passwordにCR / LFは含められません",
		},
		{
			name: "パスワードがbcryptの上限を超えるとき",
			body: strings.Replace(validRoster, "seed-password", strings.Repeat("a", validator.PasswordMaxBytes+1), 1),
			want: "passwordはbcryptの上限である72バイト以内にしてください",
		},
		{
			name: "アカウントが1件も無いとき",
			body: `password = "seed-password"`,
			want: "[[users]] が1件もありません",
		},
		{
			name: "必須項目が空のとき",
			body: strings.Replace(validRoster, `atname = "seeduser1"`, `atname = ""`, 1),
			want: "atnameが空です",
		},
		{
			name: "覚え書きが空のとき",
			body: strings.Replace(validRoster, `note = "主な確認対象"`, `note = ""`, 1),
			want: "noteが空です",
		},
		{
			name: "知らない役割を指定したとき",
			body: strings.Replace(validRoster, `role = "main"`, `role = "mian"`, 1),
			want: "生成器が知らない役割です",
		},
		{
			// メッセージが拒否した値を名指しするのは、受理されるatnameと
			// されないatnameの違いが、ファイル上では同じに見える1文字である
			// ことがあるため。
			name: "atnameに使えない文字があるとき",
			body: strings.Replace(validRoster, `atname = "seeduser1"`, `atname = "seed-user1"`, 1),
			want: `atname "seed-user1"に使える文字は半角英数字とアンダースコアだけで、20文字以内である必要があります`,
		},
		{
			name: "atnameが長すぎるとき",
			body: strings.Replace(validRoster, `atname = "seeduser1"`, `atname = "123456789012345678901"`, 1),
			want: `atname "123456789012345678901"に使える文字は半角英数字とアンダースコアだけで、20文字以内である必要があります`,
		},
		{
			name: "役割が重複しているとき",
			body: strings.Replace(validRoster, `role = "follower"`, `role = "main"`, 1),
			want: "役割mainの [[users]] が2件以上あります",
		},
		{
			name: "atnameが重複しているとき",
			body: strings.Replace(validRoster, `atname = "seeduser2"`, `atname = "seeduser1"`, 1),
			want: `atname "seeduser1"の [[users]] が2件以上あります`,
		},
		{
			// どちらのカラムもcitextであるため、この2つは一意インデックスの
			// 同じ行に行き着く。名簿はそれを、実行がデータベースを空にした後ではなく
			// 前に告げる必要がある。
			name: "atnameが大文字小文字だけ違うとき",
			body: strings.Replace(validRoster, `atname = "seeduser2"`, `atname = "SeedUser1"`, 1),
			want: `atname "SeedUser1"の [[users]] が2件以上あります`,
		},
		{
			name: "メールアドレスが大文字小文字だけ違うとき",
			body: strings.Replace(validRoster, `email = "seeduser2@example.com"`, `email = "SeedUser1@example.com"`, 1),
			want: `email "SeedUser1@example.com"の [[users]] が2件以上あります`,
		},
		{
			name: "メールアドレスの形式が不正なとき",
			body: strings.Replace(validRoster, `email = "seeduser1@example.com"`, `email = "invalid-email"`, 1),
			want: "emailがメールアドレスの形式ではありません",
		},
		{
			// この2つは解釈するとアドレスだけが取り出され、残りは落ちるが、
			// 名簿が保存するのは書かれた文字列である。どちらから作ったアカウントも
			// サインインフォームからは送信できないアドレスを持つことになるため、
			// 名簿の側で拒否する必要がある。
			name: "メールアドレスの前後に空白があるとき",
			body: strings.Replace(validRoster, `email = "seeduser1@example.com"`, `email = "seeduser1@example.com "`, 1),
			want: "emailにはアドレスだけを書いてください",
		},
		{
			name: "メールアドレスに表示名が付いているとき",
			body: strings.Replace(validRoster, `email = "seeduser1@example.com"`, `email = "シードユーザー 1 <seeduser1@example.com>"`, 1),
			want: "emailにはアドレスだけを書いてください",
		},
		{
			name: "メールアドレスが重複しているとき",
			body: strings.Replace(validRoster, `email = "seeduser2@example.com"`, `email = "seeduser1@example.com"`, 1),
			want: `email "seeduser1@example.com"の [[users]] が2件以上あります`,
		},
		{
			name: "対応していないロケールを指定したとき",
			body: strings.Replace(validRoster, `locale = "ja"`, `locale = "fr"`, 1),
			want: `locale "fr"は対応していない言語です`,
		},
		{
			name: "タイムゾーンがIANAの名前でないとき",
			body: strings.Replace(validRoster, `time_zone = "Asia/Tokyo"`, `time_zone = "Asia/Tkyo"`, 1),
			want: `time_zone "Asia/Tkyo"はIANAのタイムゾーン名として読み込めません`,
		},
		{
			// "Local" はエラー無く読み込め、実行中のマシンの設定に解決される。
			// 名簿が名指ししてはならない唯一のタイムゾーンである。
			name: "タイムゾーンにLocalを指定したとき",
			body: strings.Replace(validRoster, `time_zone = "Asia/Tokyo"`, `time_zone = "Local"`, 1),
			want: `time_zoneに"Local"は指定できません`,
		},
		{
			name: "定義されていないフィーチャーフラグを指定したとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = ["go_exmaple"]`, 1),
			want: "定義されていないフィーチャーフラグです",
		},
		{
			name: "フィーチャーフラグが重複しているとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = ["go_example", "go_example"]`, 1),
			want: "2回以上指定されています",
		},
		{
			name: "feature_flagsに知らない文字列を書いたとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = "every"`, 1),
			want: "feature_flags",
		},
		{
			name: "feature_flagsの配列要素が文字列でないとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = [1]`, 1),
			want: "feature_flagsの要素はフィーチャーフラグ名の文字列である必要があります",
		},
		{
			name: "feature_flagsが文字列でも配列でもないとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = 1`, 1),
			want: `feature_flagsは"all"かフィーチャーフラグ名の配列である必要があります`,
		},
		{
			name: "feature_flagsが無いとき",
			body: strings.Replace(validRoster, "feature_flags = \"all\"\n", "", 1),
			want: "feature_flagsがありません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := loadUserRoster(writeRoster(t, tt.body))
			if err == nil {
				t.Fatal("エラーを期待したがnilだった")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("エラーが%qを含むことを期待したが%qだった", tt.want, err)
			}
		})
	}
}

// TestLoadUserRosterRejectsMissingRoleは、値の書き換えでは作れない唯一の
// 不正な名簿。役割ごと取り除く必要があるため。これは、そうしなければ実行が
// データベースを空にした後に踏む失敗である。
//
// 取り除くのはファイルが最後に書いている1件であるため、以下で名指しする役割は
// 特定の役割ではなくvalidRosterの末尾の役割になる。
func TestLoadUserRosterRejectsMissingRole(t *testing.T) {
	t.Parallel()

	body := validRoster[:strings.LastIndex(validRoster, "[[users]]")]

	_, err := loadUserRoster(writeRoster(t, body))
	if err == nil {
		t.Fatal("役割が欠けているときのエラーを期待したがnilだった")
	}
	if !strings.Contains(err.Error(), string(roleDiscarded)) {
		t.Errorf("エラーが役割%sを名指しすることを期待したが%qだった", roleDiscarded, err)
	}
}

// TestLoadUserRosterAcceptsExampleFileは公開リポジトリが持つファイルを確認
// する。開発者がseed-users.tomlへコピーするのはこれであり、シードへ足した役割を
// 書き込む先でもある。ここで読み込むことが、それが行われていないときにそう告げる
// ことになる。
func TestLoadUserRosterAcceptsExampleFile(t *testing.T) {
	t.Parallel()

	roster, err := loadUserRoster(filepath.Join("..", "..", rosterExamplePath))
	if err != nil {
		t.Fatalf("%sの読み込みに失敗: %v", rosterExamplePath, err)
	}

	if len(roster.users) != len(allSeedRoles) {
		t.Fatalf("見本のアカウントが%d件であることを期待したが%d件だった", len(allSeedRoles), len(roster.users))
	}
	if err := auth.CheckPassword(roster.passwordDigest, "password"); err != nil {
		t.Errorf("見本のパスワードが%qであることを期待したが一致しなかった: %v", "password", err)
	}

	usersByRole := make(map[seedRole]rosterUser, len(roster.users))
	for _, user := range roster.users {
		usersByRole[user.role] = user
	}

	// 各役割は、エクスポートが読むために名簿が持つ2つの事実 (ロケールと
	// タイムゾーン) と、画面のGo版が隠れているフラグを持つかどうかで違っている。
	// ここでそれらを確認することが、見本が今も生成器の想定するアカウントを記述して
	// いることの確認になる。
	tests := []struct {
		role     seedRole
		locale   string
		timeZone string
		allFlags bool
	}{
		{role: roleMain, locale: "ja", timeZone: "Asia/Tokyo", allFlags: true},
		{role: roleFollower, locale: "ja", timeZone: "Asia/Tokyo", allFlags: false},
		{role: roleEnglish, locale: "en", timeZone: "Etc/UTC", allFlags: true},
		{role: roleNewcomer, locale: "ja", timeZone: "Asia/Tokyo", allFlags: true},
		{role: roleDiscarded, locale: "ja", timeZone: "Asia/Tokyo", allFlags: false},
	}

	for _, tt := range tests {
		user, ok := usersByRole[tt.role]
		if !ok {
			t.Errorf("見本に役割%sのアカウントが無い", tt.role)

			continue
		}
		if user.locale != tt.locale || user.timeZone != tt.timeZone {
			t.Errorf("見本の%sが%s / %sであることを期待したが%s / %sだった", tt.role, tt.locale, tt.timeZone, user.locale, user.timeZone)
		}
		if tt.allFlags && !slices.Equal(user.featureFlags, model.AllFeatureFlagNames) {
			t.Errorf("見本の%sが全フィーチャーフラグを持つことを期待したが%vだった", tt.role, user.featureFlags)
		}
		if !tt.allFlags && len(user.featureFlags) != 0 {
			t.Errorf("見本の%sがフィーチャーフラグを持たないことを期待したが%vだった", tt.role, user.featureFlags)
		}
	}
}

// writeRosterはテスト専用のディレクトリへ名簿ファイルを書き、そのパスを返す。
func writeRoster(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "seed-users.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("名簿の書き込みに失敗: %v", err)
	}

	return path
}
