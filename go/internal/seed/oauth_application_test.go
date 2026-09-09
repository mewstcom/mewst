package seed

import (
	"context"
	"testing"

	"github.com/google/uuid"

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

	applicationID, err := createOauthApplication(ctx, tx)
	if err != nil {
		t.Fatalf("OAuth アプリケーションの作成に失敗: %v", err)
	}

	var (
		id           uuid.UUID
		name         string
		secret       string
		redirectURI  string
		scopes       string
		confidential bool
	)

	// The row is read back by the uid the application looks it up with, so
	// what is checked below is the row a running Mewst would have found.
	//
	// [Ja] 行は、アプリケーションがそれを引くのに使う uid で読み戻す。以下で確認する
	// のは、動作している Mewst が見つけたであろう行になる。
	if err := tx.QueryRowContext(ctx, `
		SELECT id, name, secret, redirect_uri, scopes, confidential
		FROM oauth_applications
		WHERE uid = $1
	`, model.MewstWebUID).Scan(&id, &name, &secret, &redirectURI, &scopes, &confidential); err != nil {
		t.Fatalf("作成した OAuth アプリケーションの取得に失敗: %v", err)
	}

	// The returned id is what every generated post is attributed to, so it has
	// to be the id of the row that was written and not a zero value the post
	// generator would carry into a foreign key.
	//
	// [Ja] 返された id は、生成されるすべてのポストの帰属先になるため、書き込まれた
	// 行の id である必要がある。ゼロ値であれば、ポストの生成器がそれを外部キーとして
	// 持ち回ることになる。
	if applicationID != model.OauthApplicationID(id) {
		t.Errorf("返された id = %s, want %s", applicationID, id)
	}

	// The row has to carry the name and the redirect URI the source names.
	// Both are written only here, and a redirect URI that did not reach the
	// row would not show itself until an authorization was refused at the
	// screen.
	//
	// [Ja] 行は、ソースが名指しする name と redirect_uri を持っている必要がある。
	// どちらもここでしか書き込まれず、redirect_uri が行へ届かなかった場合、それは
	// 画面で認可が拒否されるまで現れない。
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

	// The seed omits scopes and confidential from the insert, so both values
	// have to come from the schema defaults.
	//
	// [Ja] シードは INSERT で scopes と confidential を省略するため、両方の値が
	// スキーマの既定値から設定される必要がある。
	if scopes != "" {
		t.Errorf("scopes = %q, want empty", scopes)
	}
	if !confidential {
		t.Error("confidential = false, want true")
	}
}
