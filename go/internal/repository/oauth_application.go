package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// OauthApplicationRepository はOAuthアプリケーションのリポジトリ
type OauthApplicationRepository struct {
	q *query.Queries
}

// NewOauthApplicationRepository はOauthApplicationRepositoryを生成する
func NewOauthApplicationRepository(q *query.Queries) *OauthApplicationRepository {
	return &OauthApplicationRepository{q: q}
}

// WithTx はトランザクションを設定したOauthApplicationRepositoryを返す
func (r *OauthApplicationRepository) WithTx(tx *sql.Tx) *OauthApplicationRepository {
	return &OauthApplicationRepository{q: r.q.WithTx(tx)}
}

// FindByUID はuidでOAuthアプリケーションを取得する
func (r *OauthApplicationRepository) FindByUID(ctx context.Context, uid string) (*model.OauthApplication, error) {
	row, err := r.q.GetOauthApplicationByUID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toOauthApplicationModel(row), nil
}

// toOauthApplicationModel converts a query.OauthApplication row into a model.OauthApplication.
// [Ja] toOauthApplicationModel は query.OauthApplication を model.OauthApplication に変換する
// パッケージ非公開の自由関数。
func toOauthApplicationModel(row query.OauthApplication) *model.OauthApplication {
	return &model.OauthApplication{
		ID:           model.OauthApplicationID(row.ID),
		Name:         row.Name,
		UID:          row.Uid,
		Secret:       row.Secret,
		RedirectURI:  row.RedirectUri,
		Scopes:       row.Scopes,
		Confidential: row.Confidential,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
