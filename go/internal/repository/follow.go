package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// FollowRepositoryはフォローのリポジトリ。
type FollowRepository struct {
	q *query.Queries
}

// NewFollowRepositoryはFollowRepositoryを生成する。
func NewFollowRepository(q *query.Queries) *FollowRepository {
	return &FollowRepository{q: q}
}

// WithTxはトランザクションを設定したFollowRepositoryを返す。
func (r *FollowRepository) WithTx(tx *sql.Tx) *FollowRepository {
	return &FollowRepository{q: r.q.WithTx(tx)}
}

// ListByTargetProfileIDは指定プロフィールをtargetとするfollowを返す。
// 各followのSourceProfileIDがそのプロフィールのフォロワーであり、fanoutは
// これを1クエリで取得して配信先を列挙できる (フォロワーごとのN+1を避ける)。
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

// toFollowModelはquery.Followをmodel.Followに変換する。
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
