package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// FeatureFlagRepository is the repository for feature flags.
// [Ja] FeatureFlagRepository はフィーチャーフラグのリポジトリ。
type FeatureFlagRepository struct {
	q *query.Queries
}

// NewFeatureFlagRepository creates a FeatureFlagRepository.
// [Ja] NewFeatureFlagRepository は FeatureFlagRepository を生成する。
func NewFeatureFlagRepository(q *query.Queries) *FeatureFlagRepository {
	return &FeatureFlagRepository{q: q}
}

// WithTx returns a new FeatureFlagRepository bound to the transaction.
// [Ja] WithTx はトランザクションを設定した FeatureFlagRepository を返す。
func (r *FeatureFlagRepository) WithTx(tx *sql.Tx) *FeatureFlagRepository {
	return &FeatureFlagRepository{q: r.q.WithTx(tx)}
}

// IsEnabledForActor reports whether the flag is enabled for the given actor.
// [Ja] IsEnabledForActor は指定 actor に対してフラグが有効かどうかを返す。
func (r *FeatureFlagRepository) IsEnabledForActor(ctx context.Context, actorID model.ActorID, name model.FeatureFlagName) (bool, error) {
	return r.q.IsFeatureFlagEnabledForActor(ctx, query.IsFeatureFlagEnabledForActorParams{
		ActorID: uuid.NullUUID{UUID: uuid.UUID(actorID), Valid: true},
		Name:    string(name),
	})
}

// IsEnabledForDevice reports whether the flag is enabled via a device_token or via the
// actor resolved from a session token. Both lookups are evaluated in a single query.
//
// [Ja] IsEnabledForDevice は device_token またはセッショントークン経由で解決した actor で
// フラグが有効かどうかを返す。両方の判定を 1 クエリで評価する。
func (r *FeatureFlagRepository) IsEnabledForDevice(ctx context.Context, deviceToken string, sessionToken string, name model.FeatureFlagName) (bool, error) {
	return r.q.IsFeatureFlagEnabledForDevice(ctx, query.IsFeatureFlagEnabledForDeviceParams{
		DeviceToken: sql.NullString{String: deviceToken, Valid: deviceToken != ""},
		Token:       sessionToken,
		Name:        string(name),
	})
}
