package seed

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// TestCreateOauthApplication verifies that a run leaves behind the application
// row every generated post is attributed to.
//
// posts.oauth_application_id is NOT NULL, so this row is what stands between
// the post generators and a run that empties the database and then cannot
// write a single post back into it.
//
// [Ja] TestCreateOauthApplication は、実行が、生成されるすべてのポストの帰属先と
// なるアプリケーションの行を残すことを検証する。
//
// posts.oauth_application_id は NOT NULL であり、この行は、ポストの生成器と、
// データベースを空にした後で 1 件のポストも書き戻せない実行との間に立つものになる。
func TestCreateOauthApplication(t *testing.T) {
	t.Parallel()

	// The uid is fixed and has a unique index on it, and other test packages
	// commit and delete a row with the same uid. The lock is what serializes
	// this test against them: packages run as separate processes against one
	// database, so a row another package has committed would collide with the
	// insert below.
	//
	// [Ja] uid は固定で、そこには一意インデックスがある。他のテストパッケージも
	// 同じ uid の行をコミットしては削除する。このロックが、それらとの直列化を行う。
	// パッケージは 1 つのデータベースに対して別プロセスで実行されるため、他の
	// パッケージがコミットした行は以下の INSERT と衝突する。
	testutil.AcquireMewstWebLock(t)

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	if err := createOauthApplication(ctx, tx); err != nil {
		t.Fatalf("OAuth アプリケーションの作成に失敗: %v", err)
	}

	var (
		name         string
		secret       string
		redirectURI  string
		scopes       string
		confidential bool
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT name, secret, redirect_uri, scopes, confidential
		FROM oauth_applications
		WHERE uid = $1
	`, model.MewstWebUID).Scan(&name, &secret, &redirectURI, &scopes, &confidential); err != nil {
		t.Fatalf("作成した OAuth アプリケーションの取得に失敗: %v", err)
	}

	// The row is found by the uid the application looks it up with, which is
	// the only part of it any caller depends on.
	//
	// [Ja] 行は、アプリケーションがそれを引くのに使う uid で見つける。呼び出し元が
	// 依存しているのはその 1 点だけであるため。
	if name != mewstWebApplicationName {
		t.Errorf("name = %q, want %q", name, mewstWebApplicationName)
	}
	if redirectURI != mewstWebRedirectURI {
		t.Errorf("redirect_uri = %q, want %q", redirectURI, mewstWebRedirectURI)
	}

	// The secret is generated per run rather than written into the source, so
	// what can be held here is that one was generated at all.
	//
	// [Ja] シークレットはソースへ書かず実行ごとに生成するため、ここで固定できるのは
	// 生成されたということまでになる。
	if secret == "" {
		t.Error("secret が空。生成したシークレットが書き込まれていない")
	}

	// The Rails side leaves both of these at their defaults, and a
	// development database the seed built has to describe the same
	// application as one bin/rails db:seed built.
	//
	// [Ja] Rails 側はこの 2 つを既定値のままにしている。シードが作った開発用
	// データベースは、bin/rails db:seed が作ったそれと同じアプリケーションを記述して
	// いる必要がある。
	if scopes != "" {
		t.Errorf("scopes = %q, want empty", scopes)
	}
	if !confidential {
		t.Error("confidential = false, want true")
	}
}
