package seed

import (
	"strings"
	"testing"
)

// TestRequireDevEnvironment_AllowsOnlyDev verifies that a run is refused under
// every APP_ENV but dev, an unset one included. The seed discards every
// managed row, so the check is what stands between that and a database the
// run was never meant to reach.
//
// [Ja] TestRequireDevEnvironment_AllowsOnlyDev は、dev 以外のすべての APP_ENV で
// 実行が拒否されること、未設定もそこに含まれることを検証する。シードは管理対象の
// 行をすべて破棄するため、この検査は、それと、実行が辿り着くはずのなかった
// データベースとの間に立つものになる。
func TestRequireDevEnvironment_AllowsOnlyDev(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  string
		// wantErrContains is the part of the message a developer needs in
		// order to see which environment was refused and which one is
		// required.
		//
		// [Ja] wantErrContains は、どの環境が拒否され、どの環境が必要なのかを
		// 開発者が読み取るために要るメッセージの部分。
		wantErrContains string
	}{
		{
			name: "dev では実行できる",
			env:  "dev",
		},
		{
			// config.Load reads an unset APP_ENV as dev, so the guard has to
			// be applied to the raw value as well: a run that never named an
			// environment would otherwise be taken for a development one.
			//
			// [Ja] config.Load は未設定の APP_ENV を dev として読むため、ガードは
			// 生の値に対しても適用する必要がある。そうしないと、環境を一度も名指し
			// しなかった実行が開発環境のものとみなされる。
			name:            "未設定では実行できない",
			env:             "",
			wantErrContains: "APP_ENV が設定されていません",
		},
		{
			name:            "prod では実行できない",
			env:             "prod",
			wantErrContains: "APP_ENV=prod では実行できません",
		},
		{
			// The test environment is refused like any other: `make test`
			// runs with APP_ENV=test against a database the test suite is
			// working in.
			//
			// [Ja] テスト環境も他と同じく拒否する。`make test` は APP_ENV=test で
			// 実行され、その対象はテストスイートが作業中のデータベースであるため。
			name:            "test では実行できない",
			env:             "test",
			wantErrContains: "APP_ENV=test では実行できません",
		},
		{
			name:            "大文字違いの dev では実行できない",
			env:             "DEV",
			wantErrContains: "APP_ENV=DEV では実行できません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := requireDevEnvironment(tt.env)

			if tt.wantErrContains == "" {
				if err != nil {
					t.Fatalf("requireDevEnvironment(%q) がエラーを返した: %v", tt.env, err)
				}

				return
			}

			if err == nil {
				t.Fatalf("requireDevEnvironment(%q) がエラーを返さなかった", tt.env)
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("requireDevEnvironment(%q) のエラーが %q を含むことを期待したが %q だった", tt.env, tt.wantErrContains, err)
			}
			// Whichever environment was refused, the message has to name the
			// one that is allowed. A developer who is told only that this one
			// is wrong is left to guess what to set instead.
			//
			// [Ja] どの環境が拒否された場合でも、メッセージは許可されている環境を
			// 名指しする必要がある。この環境が誤りであることだけを告げられた開発者は、
			// 代わりに何を設定すればよいのかを推測することになる。
			if want := "APP_ENV=dev"; !strings.Contains(err.Error(), want) {
				t.Errorf("requireDevEnvironment(%q) のエラーが %q を含むことを期待したが %q だった", tt.env, want, err)
			}
		})
	}
}
