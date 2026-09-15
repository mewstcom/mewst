package seed

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// TestCreateOauthApplicationは、実行が、生成されるすべてのポストの帰属先と
// なるアプリケーションの行を残すことを検証する。
//
// posts.oauth_application_idはNOT NULLであり、この行は、ポストの生成器と、
// データベースを空にした後で1件のポストも書き戻せない実行との間に立つものになる。
func TestCreateOauthApplication(t *testing.T) {
	t.Parallel()

	// uidは固定で、そこには一意インデックスがある。他のテストパッケージも
	// 同じuidの行をコミットしては削除する。このロックが、それらとの直列化を行う。
	// パッケージは1つのデータベースに対して別プロセスで実行されるため、他の
	// パッケージがコミットした行は以下のINSERTと衝突する。
	testutil.AcquireMewstWebLock(t)

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	applicationID, err := createOauthApplication(ctx, tx)
	if err != nil {
		t.Fatalf("OAuthアプリケーションの作成に失敗: %v", err)
	}

	var (
		id           uuid.UUID
		name         string
		secret       string
		redirectURI  string
		scopes       string
		confidential bool
	)

	// 行は、アプリケーションがそれを引くのに使うuidで読み戻す。以下で確認する
	// のは、動作しているMewstが見つけたであろう行になる。
	if err := tx.QueryRowContext(ctx, `
		SELECT id, name, secret, redirect_uri, scopes, confidential
		FROM oauth_applications
		WHERE uid = $1
	`, model.MewstWebUID).Scan(&id, &name, &secret, &redirectURI, &scopes, &confidential); err != nil {
		t.Fatalf("作成したOAuthアプリケーションの取得に失敗: %v", err)
	}

	// 返されたidは、生成されるすべてのポストの帰属先になるため、書き込まれた
	// 行のidである必要がある。ゼロ値であれば、ポストの生成器がそれを外部キーとして
	// 持ち回ることになる。
	if applicationID != model.OauthApplicationID(id) {
		t.Errorf("返されたid = %s、期待値 = %s", applicationID, id)
	}

	// 行は、ソースが名指しするnameとredirect_uriを持っている必要がある。
	// どちらもここでしか書き込まれず、redirect_uriが行へ届かなかった場合、それは
	// 画面で認可が拒否されるまで現れない。
	if name != mewstWebApplicationName {
		t.Errorf("name = %q、期待値 = %q", name, mewstWebApplicationName)
	}
	if redirectURI != mewstWebRedirectURI {
		t.Errorf("redirect_uri = %q、期待値 = %q", redirectURI, mewstWebRedirectURI)
	}

	// シークレットはソースへ書かず実行ごとに生成するため、ここで固定できるのは
	// 生成されたということまでになる。
	if secret == "" {
		t.Error("secretが空。生成したシークレットが書き込まれていない")
	}

	// シードはINSERTでscopesとconfidentialを省略するため、両方の値が
	// スキーマの既定値から設定される必要がある。
	if scopes != "" {
		t.Errorf("scopes = %q、空を期待", scopes)
	}
	if !confidential {
		t.Error("confidential = false、期待値 = true")
	}
}
