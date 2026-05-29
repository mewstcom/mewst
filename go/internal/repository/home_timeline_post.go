package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// HomeTimelinePostRepository is the repository for home timeline posts.
// [Ja] HomeTimelinePostRepository はホームタイムライン投稿のリポジトリ。
type HomeTimelinePostRepository struct {
	q *query.Queries
}

// NewHomeTimelinePostRepository creates a HomeTimelinePostRepository.
// [Ja] NewHomeTimelinePostRepository は HomeTimelinePostRepository を生成する。
func NewHomeTimelinePostRepository(q *query.Queries) *HomeTimelinePostRepository {
	return &HomeTimelinePostRepository{q: q}
}

// WithTx returns a HomeTimelinePostRepository bound to the given transaction.
// [Ja] WithTx はトランザクションを設定した HomeTimelinePostRepository を返す。
func (r *HomeTimelinePostRepository) WithTx(tx *sql.Tx) *HomeTimelinePostRepository {
	return &HomeTimelinePostRepository{q: r.q.WithTx(tx)}
}

// CreateHomeTimelinePostInput is the input for creating a home timeline post.
// [Ja] CreateHomeTimelinePostInput はホームタイムライン投稿作成の入力パラメータ。
type CreateHomeTimelinePostInput struct {
	ProfileID   model.ProfileID
	PostID      model.PostID
	PublishedAt time.Time
}

// Create adds a post to a profile's home timeline idempotently: calling it
// again with the same (profile_id, post_id) returns the existing row instead
// of creating a duplicate or erroring.
//
// [Ja] Create は投稿をプロフィールのホームタイムラインに冪等に追加する。同じ
// (profile_id, post_id) で再度呼んでも、重複作成やエラーにはならず既存行を返す。
func (r *HomeTimelinePostRepository) Create(ctx context.Context, input CreateHomeTimelinePostInput) (*model.HomeTimelinePost, error) {
	row, err := r.q.CreateHomeTimelinePost(ctx, query.CreateHomeTimelinePostParams{
		ProfileID:   uuid.UUID(input.ProfileID),
		PostID:      uuid.UUID(input.PostID),
		PublishedAt: input.PublishedAt,
	})
	if err != nil {
		return nil, err
	}
	return toHomeTimelinePostModel(row), nil
}

// toHomeTimelinePostModel converts a query.HomeTimelinePost row into a model.HomeTimelinePost.
// [Ja] toHomeTimelinePostModel は query.HomeTimelinePost を model.HomeTimelinePost に変換する。
func toHomeTimelinePostModel(row query.HomeTimelinePost) *model.HomeTimelinePost {
	return &model.HomeTimelinePost{
		ID:          model.HomeTimelinePostID(row.ID),
		ProfileID:   model.ProfileID(row.ProfileID),
		PostID:      model.PostID(row.PostID),
		PublishedAt: row.PublishedAt,
	}
}
