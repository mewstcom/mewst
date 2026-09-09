package seed

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
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

			err := requireDevEnvironment(tt.env, truncatesEveryManagedTable)

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

// TestGenerateSeedData exercises the actual generator sequence so that its
// cross-generator contracts stay visible: link-card posts reach the home
// timeline, and exports include every kept post while remaining in the state
// assigned to each role.
//
// [Ja] TestGenerateSeedData は実際の生成順序を通し、生成器をまたぐ契約を見える形に
// 保つ。リンクカードのポストがホームタイムラインへ届き、エクスポートが保持ポストの
// すべてを含みながら、各役割へ割り当てられた状態で残ることを検証する。
func TestGenerateSeedData(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	accounts, err := generateSeedData(ctx, tx, newTestRoster(t))
	if err != nil {
		t.Fatalf("シードデータの生成に失敗: %v", err)
	}

	assertStoredLinks(t, ctx, tx, accounts)
	posts := readLinkPosts(t, ctx, tx, accounts)
	assertLinkPostsMatchRoster(t, posts, accounts)

	main, err := accountForRole(accounts, roleMain)
	if err != nil {
		t.Fatalf("main のアカウントの取得に失敗: %v", err)
	}
	follower, err := accountForRole(accounts, roleFollower)
	if err != nil {
		t.Fatalf("follower のアカウントの取得に失敗: %v", err)
	}

	timeline := readHomeTimeline(t, ctx, tx, main.profile.ID)
	followerCards := 0
	for _, post := range posts {
		if post.profileID != follower.profile.ID {
			continue
		}
		followerCards++

		publishedAt, ok := timeline[post.postID]
		if !ok {
			t.Errorf("follower のリンク %s を持つポストが main のホームタイムラインにない", post.canonicalURL)
			continue
		}
		if !publishedAt.Equal(post.publishedAt) {
			t.Errorf("リンク %s のタイムラインの公開日時 = %v, want %v", post.canonicalURL, publishedAt, post.publishedAt)
		}
	}
	if followerCards == 0 {
		t.Fatal("ホームタイムラインで検証する follower のカード付きポストがない")
	}

	assertGeneratedExports(t, ctx, tx, accounts)
}

// assertGeneratedExports verifies the role-to-status assignment independently
// of exportStates and compares each non-terminal export's snapshot with all
// posts its profile keeps after the full generator sequence.
//
// [Ja] assertGeneratedExports は、役割と状態の対応を exportStates から独立して検証し、
// 終端状態でない各エクスポートの snapshot を、すべての生成器を通した後にその
// プロフィールが保持するポストと突き合わせる。
func assertGeneratedExports(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	accounts []seedAccount,
) {
	t.Helper()

	wantStatuses := map[seedRole]model.ExportStatus{
		roleMain:     model.ExportStatusFailed,
		roleFollower: model.ExportStatusQueued,
		roleEnglish:  model.ExportStatusStarted,
	}

	for _, role := range allSeedRoles {
		account, err := accountForRole(accounts, role)
		if err != nil {
			t.Fatalf("役割 %s のアカウントの取得に失敗: %v", role, err)
		}

		exports := readExports(t, ctx, tx, account.profile.ID)
		wantStatus, holds := wantStatuses[role]
		if !holds {
			if len(exports) != 0 {
				t.Errorf("役割 %s のエクスポートの件数 = %d, want 0", role, len(exports))
			}
			continue
		}

		if len(exports) != 1 {
			t.Fatalf("役割 %s のエクスポートの件数 = %d, want 1", role, len(exports))
		}

		export := exports[0]
		if export.status != wantStatus {
			t.Errorf("役割 %s のエクスポートの状態 = %s, want %s", role, export.status, wantStatus)
		}

		assertExportStateFields(t, role, export)
		assertExportSnapshot(t, ctx, tx, role, export, account.profile.ID)
	}
}
