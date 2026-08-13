package model

import "time"

// Feature flag name constants. Add a constant here when introducing a new flag.
// [Ja] フィーチャーフラグ名の定数。新しいフラグを追加するときはここに定数を追加する。
const (
	// FeatureFlagExample is an unused constant kept as a naming-convention example.
	// [Ja] FeatureFlagExample は命名規則の例として残している未使用の定数。
	FeatureFlagExample FeatureFlagName = "go_example"

	// FeatureFlagExport gates the export feature (the settings export screen and
	// the zip download).
	//
	// [Ja] FeatureFlagExport はエクスポート機能 (設定のエクスポート画面と zip の
	// ダウンロード) をゲートする。
	FeatureFlagExport FeatureFlagName = "go_export"
)

// FeatureFlag is the domain model for a feature flag.
// [Ja] FeatureFlag はフィーチャーフラグのドメインモデル。
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	ActorID     *ActorID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
