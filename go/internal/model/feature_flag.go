package model

import "time"

// フィーチャーフラグ名の定数。新しいフラグを追加するときは、ここに定数を追加し、
// AllFeatureFlagNamesにも追加する。
const (
	// FeatureFlagExampleは命名規則の例として残している未使用の定数。
	FeatureFlagExample FeatureFlagName = "go_example"
)

// AllFeatureFlagNamesは上で定義した全フラグの一覧。Goは定数グループの
// メンバーを列挙できないため、フラグを追加する人が必ず目にする定数のすぐ隣で
// 手作業で維持する。
var AllFeatureFlagNames = []FeatureFlagName{
	FeatureFlagExample,
}

// FeatureFlagはフィーチャーフラグのドメインモデル。
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	ActorID     *ActorID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
