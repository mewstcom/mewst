package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// PostLinkRepository is the repository for post-link associations.
// [Ja] PostLinkRepository は投稿とリンクの関連付けのリポジトリ。
type PostLinkRepository struct {
	q *query.Queries
}

// NewPostLinkRepository creates a PostLinkRepository.
// [Ja] NewPostLinkRepository は PostLinkRepository を生成する。
func NewPostLinkRepository(q *query.Queries) *PostLinkRepository {
	return &PostLinkRepository{q: q}
}

// WithTx returns a PostLinkRepository bound to the given transaction.
// [Ja] WithTx はトランザクションを設定した PostLinkRepository を返す。
func (r *PostLinkRepository) WithTx(tx *sql.Tx) *PostLinkRepository {
	return &PostLinkRepository{q: r.q.WithTx(tx)}
}

// CreatePostLinkInput is the input for creating a post-link association.
// [Ja] CreatePostLinkInput は投稿とリンクの関連付け作成の入力パラメータ。
type CreatePostLinkInput struct {
	PostID model.PostID
	LinkID model.LinkID
}

// Create inserts a new post-link association.
// [Ja] Create は投稿とリンクの関連付けを作成する。
func (r *PostLinkRepository) Create(ctx context.Context, input CreatePostLinkInput) (*model.PostLink, error) {
	row, err := r.q.CreatePostLink(ctx, query.CreatePostLinkParams{
		PostID: uuid.UUID(input.PostID),
		LinkID: uuid.UUID(input.LinkID),
	})
	if err != nil {
		return nil, err
	}
	return toPostLinkModel(row), nil
}

// toPostLinkModel converts a query.PostLink row into a model.PostLink.
// [Ja] toPostLinkModel は query.PostLink を model.PostLink に変換するパッケージ非公開の自由関数。
func toPostLinkModel(row query.PostLink) *model.PostLink {
	return &model.PostLink{
		ID:        model.PostLinkID(row.ID),
		PostID:    model.PostID(row.PostID),
		LinkID:    model.LinkID(row.LinkID),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
