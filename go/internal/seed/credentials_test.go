package seed

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// TestDevCredentialsは、サインインできる役割のそれぞれが、その役割の項目が
// 持つアドレスと、名簿が共有しているパスワードで応答されることを検証する。
func TestDevCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		role      string
		wantEmail string
	}{
		{role: "main", wantEmail: "seeduser1@example.com"},
		{role: "follower", wantEmail: "seeduser2@example.com"},
		{role: "english", wantEmail: "seeduser3@example.com"},
		{role: "newcomer", wantEmail: "seeduser4@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			t.Parallel()

			path := writeRoster(t, validRoster)

			credentials, err := devCredentials(devEnvironment, path, tt.role)
			if err != nil {
				t.Fatalf("資格情報の取得に失敗: %v", err)
			}

			if credentials.Email != tt.wantEmail {
				t.Errorf("devCredentials()のemail = %q、期待値 = %q", credentials.Email, tt.wantEmail)
			}
			if want := "seed-password"; credentials.Password != want {
				t.Errorf("devCredentials()のpassword = %q、期待値 = %q", credentials.Password, want)
			}
		})
	}
}

// TestDevCredentialsAnswersWithThePasswordTheAccountsHoldは、名簿を読む
// 2つの経路を結び付ける。ここで報告されるパスワードは、同じファイルから実行が
// 書き込んだアカウントをサインインさせるものである必要がある。それを確かめるものは
// 他に無く、2つの経路はパスワードへの辿り着き方が異なる。一方はハッシュ化し、
// 他方はそのまま運ぶ。
func TestDevCredentialsAnswersWithThePasswordTheAccountsHold(t *testing.T) {
	t.Parallel()

	path := writeRoster(t, validRoster)

	roster, err := loadUserRoster(path)
	if err != nil {
		t.Fatalf("名簿の読み込みに失敗: %v", err)
	}

	credentials, err := devCredentials(devEnvironment, path, string(roleMain))
	if err != nil {
		t.Fatalf("資格情報の取得に失敗: %v", err)
	}

	if err := auth.CheckPassword(roster.passwordDigest, credentials.Password); err != nil {
		t.Errorf("報告されたパスワードがアカウントのダイジェストと一致しません: %v", err)
	}
}

// TestDevCredentialsRefusesARoleThatCannotSignInは、削除済みの役割が、
// 渡されるのではなくそのものとして拒否されることを検証する。そのプロフィールは
// 削除済みであり、削除済みプロフィールがサインインする状態は本番では起こり得ない。
func TestDevCredentialsRefusesARoleThatCannotSignIn(t *testing.T) {
	t.Parallel()

	path := writeRoster(t, validRoster)

	_, err := devCredentials(devEnvironment, path, string(roleDiscarded))
	if err == nil {
		t.Fatal("削除済みの役割が資格情報として受理された")
	}

	for _, want := range []string{"サインインできません", "main, follower, english, newcomer"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("devCredentials()のエラー = %q、%qを含むことを期待", err, want)
		}
	}
}

// TestDevCredentialsRefusesAnUnknownRoleは、名簿が持たない名前が、名簿が
// 持つ名前を添えて拒否されることを検証する。
func TestDevCredentialsRefusesAnUnknownRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role string
	}{
		{name: "知らない役割", role: "nosuchrole"},
		{name: "空の役割", role: ""},
		{name: "大文字違いの役割", role: "MAIN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writeRoster(t, validRoster)

			_, err := devCredentials(devEnvironment, path, tt.role)
			if err == nil {
				t.Fatalf("役割%qが受理された", tt.role)
			}
			if want := "main, follower, english, newcomer"; !strings.Contains(err.Error(), want) {
				t.Errorf("devCredentials()のエラー = %q、%qを含むことを期待", err, want)
			}
		})
	}
}

// TestDevCredentialsRefusesRunningOutsideDevelopmentは、シードの実行が課して
// いるのと同じガードがここでも効くこと、拒否が資格情報に固有の理由を述べること、
// そしてそれが名簿を読むより前に効くことを検証する。渡すパスは存在しないため、
// ファイルを名指しするエラーは、検査が遅すぎたことを意味する。
func TestDevCredentialsRefusesRunningOutsideDevelopment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  string
	}{
		{name: "APP_ENVが未設定"},
		{name: "本番環境", env: "prod"},
		{name: "テスト環境", env: "test"},
		{name: "大文字違いのdev", env: "DEV"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			missing := filepath.Join(t.TempDir(), "seed-users.toml")

			_, err := devCredentials(tt.env, missing, string(roleMain))
			if err == nil {
				t.Fatalf("APP_ENV=%qでの実行が受理された", tt.env)
			}
			if want := appEnvVar; !strings.Contains(err.Error(), want) {
				t.Errorf("devCredentials()のエラー = %q、%qを含むことを期待", err, want)
			}
			if want := readsTheRosterPassword; !strings.Contains(err.Error(), want) {
				t.Errorf("devCredentials()のエラー = %q、%qを含むことを期待", err, want)
			}
			if notWant := truncatesEveryManagedTable; strings.Contains(err.Error(), notWant) {
				t.Errorf("devCredentials()のエラー = %q、%qを含まないことを期待", err, notWant)
			}
			if strings.Contains(err.Error(), missing) {
				t.Errorf("devCredentials()のエラー = %q、名簿を読む前に拒否することを期待", err)
			}
		})
	}
}

// TestDevCredentialsRefusesARosterItCannotUseは、シードの実行が名簿に課して
// いる検査がここでも効くことを検証する。双方が読むのは同じファイルであり、一方が
// 受理して他方が拒否する名簿は、作成されなかったアカウントを報告することになる。
func TestDevCredentialsRefusesARosterItCannotUse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		roster  string
		wantMsg string
	}{
		{
			name:    "パスワードが空",
			roster:  strings.Replace(validRoster, `password = "seed-password"`, `password = ""`, 1),
			wantMsg: "passwordが空です",
		},
		{
			name:    "パスワードがbcryptの上限を超える",
			roster:  strings.Replace(validRoster, "seed-password", strings.Repeat("a", validator.PasswordMaxBytes+1), 1),
			wantMsg: "passwordはbcryptの上限である72バイト以内にしてください",
		},
		{
			name:    "役割が重複している",
			roster:  strings.Replace(validRoster, `role = "discarded"`, `role = "newcomer"`, 1),
			wantMsg: "役割newcomerの [[users]] が2件以上あります",
		},
		{
			name:    "atnameが形式に合わない",
			roster:  strings.Replace(validRoster, `atname = "seeduser1"`, `atname = "seed user1"`, 1),
			wantMsg: "atname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writeRoster(t, tt.roster)

			_, err := devCredentials(devEnvironment, path, string(roleMain))
			if err == nil {
				t.Fatal("使えない名簿が受理された")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("devCredentials()のエラー = %q、%qを含むことを期待", err, tt.wantMsg)
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("devCredentials()のエラー = %q、名簿%qを名指しすることを期待", err, path)
			}
		})
	}
}

// TestDevCredentialsRefusesAMissingRosterは、名簿が無い場合に、シードの実行と
// 同じく、コピー元の見本を添えて応答されることを検証する。
func TestDevCredentialsRefusesAMissingRoster(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "seed-users.toml")

	_, err := devCredentials(devEnvironment, missing, string(roleMain))
	if err == nil {
		t.Fatal("名簿が無い状態が受理された")
	}
	for _, want := range []string{missing, rosterExamplePath} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("devCredentials()のエラー = %q、%qを含むことを期待", err, want)
		}
	}
}
