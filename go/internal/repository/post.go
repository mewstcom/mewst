package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// PostRepository は投稿のリポジトリ
type PostRepository struct {
	q *query.Queries
}

// NewPostRepository はPostRepositoryを生成する
func NewPostRepository(q *query.Queries) *PostRepository {
	return &PostRepository{q: q}
}

// WithTx はトランザクションを設定したPostRepositoryを返す
func (r *PostRepository) WithTx(tx *sql.Tx) *PostRepository {
	return &PostRepository{q: r.q.WithTx(tx)}
}

// FindByID はIDで投稿を取得する
func (r *PostRepository) FindByID(ctx context.Context, id model.PostID) (*model.Post, error) {
	row, err := r.q.GetPostByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toPostModel(row), nil
}

// CreatePostInput は投稿作成の入力パラメータ
type CreatePostInput struct {
	ProfileID          model.ProfileID
	Content            string
	PublishedAt        time.Time
	OauthApplicationID model.OauthApplicationID
}

// Create は投稿を作成する
func (r *PostRepository) Create(ctx context.Context, input CreatePostInput) (*model.Post, error) {
	row, err := r.q.CreatePost(ctx, query.CreatePostParams{
		ProfileID:          uuid.UUID(input.ProfileID),
		Content:            input.Content,
		PublishedAt:        input.PublishedAt,
		OauthApplicationID: uuid.UUID(input.OauthApplicationID),
	})
	if err != nil {
		return nil, err
	}
	return toPostModel(row), nil
}

// toPostModel converts a query.Post row into a model.Post.
// [Ja] toPostModel は query.Post を model.Post に変換するパッケージ非公開の自由関数。
func toPostModel(row query.Post) *model.Post {
	var discardedAt *time.Time
	if row.DiscardedAt.Valid {
		discardedAt = &row.DiscardedAt.Time
	}

	return &model.Post{
		ID:                 model.PostID(row.ID),
		ProfileID:          model.ProfileID(row.ProfileID),
		Content:            row.Content,
		PublishedAt:        row.PublishedAt,
		OauthApplicationID: model.OauthApplicationID(row.OauthApplicationID),
		DiscardedAt:        discardedAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}
