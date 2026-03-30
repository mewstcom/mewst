package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/mewstcom/mewst/go/internal/query"
)

// RateLimitRepository はRate Limitのリポジトリ
type RateLimitRepository struct {
	queries *query.Queries
}

// NewRateLimitRepository はRateLimitRepositoryを生成する
func NewRateLimitRepository(db query.DBTX) *RateLimitRepository {
	return &RateLimitRepository{
		queries: query.New(db),
	}
}

// WithTx はトランザクションを設定したRateLimitRepositoryを返す
func (r *RateLimitRepository) WithTx(tx *sql.Tx) *RateLimitRepository {
	return &RateLimitRepository{
		queries: r.queries.WithTx(tx),
	}
}

// IncrementParams はRate Limitカウンターインクリメントのパラメータ
type IncrementParams struct {
	Key         string
	WindowStart time.Time
}

// IncrementResult はRate Limitカウンターインクリメントの結果
type IncrementResult struct {
	Count int32
}

// Increment はRate Limitカウンターをインクリメントする
func (r *RateLimitRepository) Increment(ctx context.Context, params IncrementParams) (*IncrementResult, error) {
	row, err := r.queries.IncrementRateLimit(ctx, query.IncrementRateLimitParams{
		Key:         params.Key,
		WindowStart: params.WindowStart,
	})
	if err != nil {
		return nil, err
	}

	return &IncrementResult{
		Count: row.Count,
	}, nil
}

// DeleteOldRecords は指定された時刻より古いRate Limitレコードを削除する
func (r *RateLimitRepository) DeleteOldRecords(ctx context.Context, cutoff time.Time) error {
	return r.queries.DeleteOldRateLimits(ctx, cutoff)
}
