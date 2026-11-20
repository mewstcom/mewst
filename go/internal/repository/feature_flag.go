package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// FeatureFlagRepositoryはフィーチャーフラグのリポジトリ。
type FeatureFlagRepository struct {
	q *query.Queries
}

// NewFeatureFlagRepositoryはFeatureFlagRepositoryを生成する。
func NewFeatureFlagRepository(q *query.Queries) *FeatureFlagRepository {
	return &FeatureFlagRepository{q: q}
}

// WithTxはトランザクションを設定したFeatureFlagRepositoryを返す。
func (r *FeatureFlagRepository) WithTx(tx *sql.Tx) *FeatureFlagRepository {
	return &FeatureFlagRepository{q: r.q.WithTx(tx)}
}

// IsEnabledForActorは指定actorに対してフラグが有効かどうかを返す。
func (r *FeatureFlagRepository) IsEnabledForActor(ctx context.Context, actorID model.ActorID, name model.FeatureFlagName) (bool, error) {
	return r.q.IsFeatureFlagEnabledForActor(ctx, query.IsFeatureFlagEnabledForActorParams{
		ActorID: uuid.NullUUID{UUID: uuid.UUID(actorID), Valid: true},
		Name:    string(name),
	})
}

// IsEnabledForDeviceはdevice_tokenまたはセッショントークン経由で解決したactorで
// フラグが有効かどうかを返す。両方の判定を1クエリで評価する。
func (r *FeatureFlagRepository) IsEnabledForDevice(ctx context.Context, deviceToken string, sessionToken string, name model.FeatureFlagName) (bool, error) {
	return r.q.IsFeatureFlagEnabledForDevice(ctx, query.IsFeatureFlagEnabledForDeviceParams{
		DeviceToken: sql.NullString{String: deviceToken, Valid: deviceToken != ""},
		Token:       sessionToken,
		Name:        string(name),
	})
}
