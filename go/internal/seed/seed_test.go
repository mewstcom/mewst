package seed

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// TestRequireDevEnvironment_AllowsOnlyDevは、dev以外のすべてのAPP_ENVで
// 実行が拒否されること、未設定もそこに含まれることを検証する。シードは管理対象の
// 行をすべて破棄するため、この検査は、それと、実行が辿り着くはずのなかった
// データベースとの間に立つものになる。
func TestRequireDevEnvironment_AllowsOnlyDev(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  string
		// wantErrContainsは、どの環境が拒否され、どの環境が必要なのかを
		// 開発者が読み取るために要るメッセージの部分。
		wantErrContains string
	}{
		{
			name: "devでは実行できる",
			env:  "dev",
		},
		{
			// config.Loadは未設定のAPP_ENVをdevとして読むため、ガードは
			// 生の値に対しても適用する必要がある。そうしないと、環境を一度も名指し
			// しなかった実行が開発環境のものとみなされる。
			name:            "未設定では実行できない",
			env:             "",
			wantErrContains: "APP_ENVが設定されていません",
		},
		{
			name:            "prodでは実行できない",
			env:             "prod",
			wantErrContains: "APP_ENV=prodでは実行できません",
		},
		{
			// テスト環境も他と同じく拒否する。`make test` はAPP_ENV=testで
			// 実行され、その対象はテストスイートが作業中のデータベースであるため。
			name:            "testでは実行できない",
			env:             "test",
			wantErrContains: "APP_ENV=testでは実行できません",
		},
		{
			name:            "大文字違いのdevでは実行できない",
			env:             "DEV",
			wantErrContains: "APP_ENV=DEVでは実行できません",
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
				t.Errorf("requireDevEnvironment(%q) のエラーが%qを含むことを期待したが%qだった", tt.env, tt.wantErrContains, err)
			}
			// どの環境が拒否された場合でも、メッセージは許可されている環境を
			// 名指しする必要がある。この環境が誤りであることだけを告げられた開発者は、
			// 代わりに何を設定すればよいのかを推測することになる。
			if want := "APP_ENV=dev"; !strings.Contains(err.Error(), want) {
				t.Errorf("requireDevEnvironment(%q) のエラーが%qを含むことを期待したが%qだった", tt.env, want, err)
			}
		})
	}
}

// TestGenerateSeedDataは実際の生成順序を通し、生成器をまたぐ契約を見える形に
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
		t.Fatalf("mainのアカウントの取得に失敗: %v", err)
	}
	follower, err := accountForRole(accounts, roleFollower)
	if err != nil {
		t.Fatalf("followerのアカウントの取得に失敗: %v", err)
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
			t.Errorf("followerのリンク%sを持つポストがmainのホームタイムラインにない", post.canonicalURL)
			continue
		}
		if !publishedAt.Equal(post.publishedAt) {
			t.Errorf("リンク%sのタイムラインの公開日時 = %v、期待値 = %v", post.canonicalURL, publishedAt, post.publishedAt)
		}
	}
	if followerCards == 0 {
		t.Fatal("ホームタイムラインで検証するfollowerのカード付きポストがない")
	}

	assertGeneratedExports(t, ctx, tx, accounts)
}

// assertGeneratedExportsは、役割と状態の対応をexportStatesから独立して検証し、
// 終端状態でない各エクスポートのsnapshotを、すべての生成器を通した後にその
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
			t.Fatalf("役割%sのアカウントの取得に失敗: %v", role, err)
		}

		exports := readExports(t, ctx, tx, account.profile.ID)
		wantStatus, holds := wantStatuses[role]
		if !holds {
			if len(exports) != 0 {
				t.Errorf("役割%sのエクスポートの件数 = %d、期待値 = 0", role, len(exports))
			}
			continue
		}

		if len(exports) != 1 {
			t.Fatalf("役割%sのエクスポートの件数 = %d、期待値 = 1", role, len(exports))
		}

		export := exports[0]
		if export.status != wantStatus {
			t.Errorf("役割%sのエクスポートの状態 = %s、期待値 = %s", role, export.status, wantStatus)
		}

		assertExportStateFields(t, role, export)
		assertExportSnapshot(t, ctx, tx, role, export, account.profile.ID)
	}
}
