package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// FollowRepository is the repository for follows.
// [Ja] FollowRepository はフォローのリポジトリ。
type FollowRepository struct {
	q *query.Queries
}

// NewFollowRepository creates a FollowRepository.
// [Ja] NewFollowRepository は FollowRepository を生成する。
func NewFollowRepository(q *query.Queries) *FollowRepository {
	return &FollowRepository{q: q}
}

// WithTx returns a FollowRepository bound to the given transaction.
// [Ja] WithTx はトランザクションを設定した FollowRepository を返す。
func (r *FollowRepository) WithTx(tx *sql.Tx) *FollowRepository {
	return &FollowRepository{q: r.q.WithTx(tx)}
}

// ListByTargetProfileID returns the follows targeting the given profile. Each
// follow's SourceProfileID is one of the profile's followers, so fanout can
// enumerate delivery targets from this in a single query (no N+1 per follower).
//
// [Ja] ListByTargetProfileID は指定プロフィールを target とする follow を返す。
// 各 follow の SourceProfileID がそのプロフィールのフォロワーであり、fanout は
// これを 1 クエリで取得して配信先を列挙できる (フォロワーごとの N+1 を避ける)。
func (r *FollowRepository) ListByTargetProfileID(ctx context.Context, targetProfileID model.ProfileID) ([]*model.Follow, error) {
	rows, err := r.q.ListFollowsByTargetProfileID(ctx, uuid.UUID(targetProfileID))
	if err != nil {
		return nil, err
	}

	follows := make([]*model.Follow, len(rows))
	for i, row := range rows {
		follows[i] = toFollowModel(row)
	}
	return follows, nil
}

// toFollowModel converts a query.Follow row into a model.Follow.
// [Ja] toFollowModel は query.Follow を model.Follow に変換する。
func toFollowModel(row query.Follow) *model.Follow {
	return &model.Follow{
		ID:              model.FollowID(row.ID),
		SourceProfileID: model.ProfileID(row.SourceProfileID),
		TargetProfileID: model.ProfileID(row.TargetProfileID),
		FollowedAt:      row.FollowedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
