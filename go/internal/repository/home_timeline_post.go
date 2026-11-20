package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// HomeTimelinePostRepositoryはホームタイムライン投稿のリポジトリ。
type HomeTimelinePostRepository struct {
	q *query.Queries
}

// NewHomeTimelinePostRepositoryはHomeTimelinePostRepositoryを生成する。
func NewHomeTimelinePostRepository(q *query.Queries) *HomeTimelinePostRepository {
	return &HomeTimelinePostRepository{q: q}
}

// WithTxはトランザクションを設定したHomeTimelinePostRepositoryを返す。
func (r *HomeTimelinePostRepository) WithTx(tx *sql.Tx) *HomeTimelinePostRepository {
	return &HomeTimelinePostRepository{q: r.q.WithTx(tx)}
}

// CreateHomeTimelinePostInputはホームタイムライン投稿作成の入力パラメータ。
type CreateHomeTimelinePostInput struct {
	ProfileID   model.ProfileID
	PostID      model.PostID
	PublishedAt time.Time
}

// Createは投稿をプロフィールのホームタイムラインに冪等に追加する。同じ
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

// toHomeTimelinePostModelはquery.HomeTimelinePostをmodel.HomeTimelinePostに変換する。
func toHomeTimelinePostModel(row query.HomeTimelinePost) *model.HomeTimelinePost {
	return &model.HomeTimelinePost{
		ID:          model.HomeTimelinePostID(row.ID),
		ProfileID:   model.ProfileID(row.ProfileID),
		PostID:      model.PostID(row.PostID),
		PublishedAt: row.PublishedAt,
	}
}
