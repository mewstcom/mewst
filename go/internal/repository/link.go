package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// LinkRepository is the repository for links.
// [Ja] LinkRepository はリンクのリポジトリ。
type LinkRepository struct {
	q *query.Queries
}

// NewLinkRepository creates a LinkRepository.
// [Ja] NewLinkRepository は LinkRepository を生成する。
func NewLinkRepository(q *query.Queries) *LinkRepository {
	return &LinkRepository{q: q}
}

// WithTx returns a LinkRepository bound to the given transaction.
// [Ja] WithTx はトランザクションを設定した LinkRepository を返す。
func (r *LinkRepository) WithTx(tx *sql.Tx) *LinkRepository {
	return &LinkRepository{q: r.q.WithTx(tx)}
}

// FindByCanonicalURL returns the link with the given canonical URL, or nil if
// none exists. Link reuse is keyed on canonical_url (it has a unique index), so
// the metadata fetcher can avoid re-creating a link for an already-known URL.
//
// [Ja] FindByCanonicalURL は指定 canonical URL のリンクを返し、存在しなければ
// nil を返す。リンクの再利用は canonical_url (unique インデックスあり) をキーに
// 行うため、メタデータ取得側は既知 URL のリンクを作り直さずに済む。
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

// CreateLinkInput is the input for creating a link.
// [Ja] CreateLinkInput はリンク作成の入力パラメータ。
type CreateLinkInput struct {
	CanonicalURL string
	Domain       string
	Title        string
	ImageURL     string
}

// Create inserts a new link.
// [Ja] Create はリンクを作成する。
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

// toLinkModel converts a query.Link row into a model.Link.
// [Ja] toLinkModel は query.Link を model.Link に変換するパッケージ非公開の自由関数。
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
