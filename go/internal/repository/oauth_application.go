package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// OauthApplicationRepositoryはOAuthアプリケーションのリポジトリ
type OauthApplicationRepository struct {
	q *query.Queries
}

// NewOauthApplicationRepositoryはOauthApplicationRepositoryを生成する
func NewOauthApplicationRepository(q *query.Queries) *OauthApplicationRepository {
	return &OauthApplicationRepository{q: q}
}

// WithTxはトランザクションを設定したOauthApplicationRepositoryを返す
func (r *OauthApplicationRepository) WithTx(tx *sql.Tx) *OauthApplicationRepository {
	return &OauthApplicationRepository{q: r.q.WithTx(tx)}
}

// FindByUIDはuidでOAuthアプリケーションを取得する
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

// toOauthApplicationModelはquery.OauthApplicationをmodel.OauthApplicationに変換する
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
