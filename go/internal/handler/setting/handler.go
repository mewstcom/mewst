// Package settingは設定メニューページのHTTPハンドラーを提供します。
package setting

import (
	"github.com/mewstcom/mewst/go/internal/config"
)

// Handlerは設定メニューページのHTTPハンドラー。
type Handler struct {
	cfg *config.Config
}

// NewHandlerは新しいHandlerを作成する。メニューは永続化を伴わない
// ナビゲーションハブのため、描画する内容はページメタデータ用のcfgで足りる。
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg: cfg,
	}
}
