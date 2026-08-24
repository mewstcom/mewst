// Package setting provides the HTTP handler for the settings menu page.
//
// [Ja] Package setting は設定メニューページの HTTP ハンドラーを提供します。
package setting

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler is the HTTP handler for the settings menu page.
//
// [Ja] Handler は設定メニューページの HTTP ハンドラー。
type Handler struct {
	cfg               *config.Config
	getSettingIndexUC *usecase.GetSettingIndexUsecase
}

// NewHandler creates a new Handler. The menu itself is a navigation hub with no
// persistence, so cfg (for page metadata) covers everything it renders on its
// own; getSettingIndexUC supplies only the feature-flag decision behind the
// export entry.
//
// [Ja] NewHandler は新しい Handler を作成する。メニュー自体は永続化を伴わない
// ナビゲーションハブのため、自力で描画する内容はページメタデータ用の cfg で足りる。
// getSettingIndexUC はエクスポート項目の背後にあるフィーチャーフラグ判定だけを担う。
func NewHandler(cfg *config.Config, getSettingIndexUC *usecase.GetSettingIndexUsecase) *Handler {
	return &Handler{
		cfg:               cfg,
		getSettingIndexUC: getSettingIndexUC,
	}
}
