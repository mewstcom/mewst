package seed

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
)

// mewstWebApplicationName is what the mewst-web row is called. It is the name
// the Rails side gives it, so that a development database the seed built and
// one bin/rails db:seed built describe the same application rather than two
// that happen to share a uid.
//
// [Ja] mewstWebApplicationName は mewst-web の行の名前。Rails 側が与えているのと
// 同じ名前にしている。シードが作った開発用データベースと bin/rails db:seed が
// 作ったそれとが、たまたま uid を共有している 2 つのアプリケーションではなく、
// 同じ 1 つのアプリケーションを記述するようにするため。
const mewstWebApplicationName = "Mewst for Web"

// mewstWebRedirectURI is where the authorization code flow would send a
// browser back to. The web front end never runs that flow — it reaches a
// profile's own posts through the session — but the column is NOT NULL, and
// example.com is the address to write where a development environment needs
// one.
//
// [Ja] mewstWebRedirectURI は、認可コードフローがブラウザを送り返す先。web
// フロントエンドはこのフローを通らず、セッション経由で自身のポストへ辿り着くが、
// カラムは NOT NULL である。開発環境がアドレスを必要とする箇所には example.com を
// 書く。
const mewstWebRedirectURI = "https://example.com/callback"

// createOauthApplication writes the application every generated post is
// attributed to.
//
// The seed writes this row rather than leaving it to bin/rails db:seed.
// posts.oauth_application_id is NOT NULL and every post the seed creates
// points at this one row, so a seed that did not create it could only run on a
// database Rails had seeded first and nobody had emptied since.
//
// The secret is generated on each run rather than written into the source. The
// application finds this row by its uid and never reads the secret by value,
// so a constant here would be a credential committed to version control that
// no caller has a use for.
//
// [Ja] createOauthApplication は、生成されるすべてのポストの帰属先となる
// アプリケーションを書き込む。
//
// この行を bin/rails db:seed に任せず、シードが書き込む。
// posts.oauth_application_id は NOT NULL であり、シードが作るすべてのポストがこの
// 1 行を指す。その行を作らないシードは、Rails が先にシードを実行し、以降誰も空に
// していないデータベースの上でしか実行できない。
//
// シークレットはソースへ書かず、実行のたびに生成する。アプリケーションはこの行を
// uid で引き、シークレットを値として読むことはない。ここに定数を置くことは、使い道の
// ある呼び出し元がいないまま、バージョン管理へ資格情報をコミットすることになる。
func createOauthApplication(ctx context.Context, tx *sql.Tx) error {
	secret, err := auth.GenerateSecureToken()
	if err != nil {
		return fmt.Errorf("OAuth アプリケーションのシークレットの生成に失敗: %w", err)
	}

	// scopes and confidential are left to the column defaults, which is what
	// the Rails side leaves them at as well.
	//
	// [Ja] scopes と confidential はカラムの既定値に任せる。Rails 側も同じく
	// 既定値のままにしている。
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO oauth_applications (name, uid, secret, redirect_uri, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, mewstWebApplicationName, model.MewstWebUID, secret, mewstWebRedirectURI); err != nil {
		return fmt.Errorf("OAuth アプリケーション %s の作成に失敗: %w", model.MewstWebUID, err)
	}

	return nil
}
