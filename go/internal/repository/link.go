package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// LinkRepositoryはリンクのリポジトリ。
type LinkRepository struct {
	q *query.Queries
}

// NewLinkRepositoryはLinkRepositoryを生成する。
func NewLinkRepository(q *query.Queries) *LinkRepository {
	return &LinkRepository{q: q}
}

// WithTxはトランザクションを設定したLinkRepositoryを返す。
func (r *LinkRepository) WithTx(tx *sql.Tx) *LinkRepository {
	return &LinkRepository{q: r.q.WithTx(tx)}
}

// FindByCanonicalURLは指定canonical URLのリンクを返し、存在しなければ
// nilを返す。リンクの再利用はcanonical_url (uniqueインデックスあり) をキーに
// 行うため、メタデータ取得側は既知URLのリンクを作り直さずに済む。
func (r *LinkRepository) FindByCanonicalURL(ctx context.Context, canonicalURL string) (*model.Link, error) {
	row, err := r.q.GetLinkByCanonicalURL(ctx, canonicalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toLinkModel(row), nil
}

// CreateLinkInputはリンク作成の入力パラメータ。
type CreateLinkInput struct {
	CanonicalURL string
	Domain       string
	Title        string
	ImageURL     string
}

// Createはリンクを作成する。
func (r *LinkRepository) Create(ctx context.Context, input CreateLinkInput) (*model.Link, error) {
	row, err := r.q.CreateLink(ctx, query.CreateLinkParams{
		CanonicalUrl: input.CanonicalURL,
		Domain:       input.Domain,
		Title:        input.Title,
		ImageUrl:     input.ImageURL,
	})
	if err != nil {
		return nil, err
	}
	return toLinkModel(row), nil
}

// toLinkModelはquery.Linkをmodel.Linkに変換するパッケージ非公開の自由関数。
func toLinkModel(row query.Link) *model.Link {
	return &model.Link{
		ID:           model.LinkID(row.ID),
		CanonicalURL: row.CanonicalUrl,
		Domain:       row.Domain,
		Title:        row.Title,
		ImageURL:     row.ImageUrl,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
