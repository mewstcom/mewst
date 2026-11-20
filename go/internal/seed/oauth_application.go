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

// mewstWebRedirectURIは、認可コードフローがブラウザを送り返す先。web
// フロントエンドはこのフローを通らず、セッション経由で自身のポストへ辿り着くが、
// カラムはNOT NULLである。開発環境がアドレスを必要とする箇所にはexample.comを
// 書く。
const mewstWebRedirectURI = "https://example.com/callback"

// createOauthApplicationは、生成されるすべてのポストの帰属先となる
// アプリケーションを書き込み、そのidを返す。
//
// idはポストの生成器が再取得せず、戻り値で渡す。すべてのポストがこの行を参照する
// ため、戻り値にすることでその依存関係をコンパイラが検査できる。アプリケーションの
// 作成より前にポストを書くように処理を並べ替えると、実行時にuidに対応する行が
// 見つからなくなる代わりに、ビルドが失敗する。
//
// posts.oauth_application_idはNOT NULLであり、生成するすべてのポストがこの行を
// 指すため、シードがこの行を書き込む。同じ実行内で作成することで、ポストが依存する
// アプリケーションデータをシード自身がすべて所有する。
//
// シークレットはソースへ書かず、実行のたびに生成する。アプリケーションはこの行を
// uidで引き、シークレットを値として読むことはない。ここに定数を置くことは、使い道の
// ある呼び出し元がいないまま、バージョン管理へ資格情報をコミットすることになる。
func createOauthApplication(ctx context.Context, tx *sql.Tx) (model.OauthApplicationID, error) {
	secret, err := auth.GenerateSecureToken()
	if err != nil {
		return model.OauthApplicationID{}, fmt.Errorf("OAuthアプリケーションのシークレットの生成に失敗: %w", err)
	}

	// scopesとconfidentialはスキーマの既定値に任せ、シード内に既定値を
	// 重複させずデータベースの契約に従う。
	var id uuid.UUID
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO oauth_applications (name, uid, secret, redirect_uri, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`, mewstWebApplicationName, model.MewstWebUID, secret, mewstWebRedirectURI).Scan(&id); err != nil {
		return model.OauthApplicationID{}, fmt.Errorf("OAuthアプリケーション%sの作成に失敗: %w", model.MewstWebUID, err)
	}

	return model.OauthApplicationID(id), nil
}
