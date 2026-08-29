package model

import "time"

// Feature flag name constants. When introducing a new flag, add a constant
// here and add it to AllFeatureFlagNames as well.
//
// [Ja] フィーチャーフラグ名の定数。新しいフラグを追加するときは、ここに定数を追加し、
// AllFeatureFlagNames にも追加する。
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

// AllFeatureFlagNames lists every flag defined above. Go cannot enumerate the
// members of a constant group, so the list is kept by hand right next to the
// constants, where whoever adds a flag is already looking.
//
// [Ja] AllFeatureFlagNames は上で定義した全フラグの一覧。Go は定数グループの
// メンバーを列挙できないため、フラグを追加する人が必ず目にする定数のすぐ隣で
// 手作業で維持する。
var AllFeatureFlagNames = []FeatureFlagName{
	FeatureFlagExample,
	FeatureFlagExport,
}

// FeatureFlag is the domain model for a feature flag.
// [Ja] FeatureFlag はフィーチャーフラグのドメインモデル。
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	ActorID     *ActorID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
