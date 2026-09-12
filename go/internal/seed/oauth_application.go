package seed

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
)

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
// attributed to, and returns its id.
//
// The id is returned rather than looked up again by the post generator. Every
// post points at this row, so returning it makes that dependency one the
// compiler sees: a generation reordered so that posts are written before the
// application exists would not build, instead of finding nothing under the
// uid at run time.
//
// The seed writes this row because posts.oauth_application_id is NOT NULL and
// every generated post points at it. Creating the row in the same run makes
// the seed own all application data its posts depend on.
//
// The secret is generated on each run rather than written into the source. The
// application finds this row by its uid and never reads the secret by value,
// so a constant here would be a credential committed to version control that
// no caller has a use for.
//
// [Ja] createOauthApplication は、生成されるすべてのポストの帰属先となる
// アプリケーションを書き込み、その id を返す。
//
// id はポストの生成器が再取得せず、戻り値で渡す。すべてのポストがこの行を参照する
// ため、戻り値にすることでその依存関係をコンパイラが検査できる。アプリケーションの
// 作成より前にポストを書くように処理を並べ替えると、実行時に uid に対応する行が
// 見つからなくなる代わりに、ビルドが失敗する。
//
// posts.oauth_application_id は NOT NULL であり、生成するすべてのポストがこの行を
// 指すため、シードがこの行を書き込む。同じ実行内で作成することで、ポストが依存する
// アプリケーションデータをシード自身がすべて所有する。
//
// シークレットはソースへ書かず、実行のたびに生成する。アプリケーションはこの行を
// uid で引き、シークレットを値として読むことはない。ここに定数を置くことは、使い道の
// ある呼び出し元がいないまま、バージョン管理へ資格情報をコミットすることになる。
func createOauthApplication(ctx context.Context, tx *sql.Tx) (model.OauthApplicationID, error) {
	secret, err := auth.GenerateSecureToken()
	if err != nil {
		return model.OauthApplicationID{}, fmt.Errorf("OAuth アプリケーションのシークレットの生成に失敗: %w", err)
	}

	// scopes and confidential use their schema defaults so the seed follows
	// the database contract instead of duplicating those defaults.
	//
	// [Ja] scopes と confidential はスキーマの既定値に任せ、シード内に既定値を
	// 重複させずデータベースの契約に従う。
	var id uuid.UUID
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO oauth_applications (name, uid, secret, redirect_uri, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`, mewstWebApplicationName, model.MewstWebUID, secret, mewstWebRedirectURI).Scan(&id); err != nil {
		return model.OauthApplicationID{}, fmt.Errorf("OAuth アプリケーション %s の作成に失敗: %w", model.MewstWebUID, err)
	}

	return model.OauthApplicationID(id), nil
}
