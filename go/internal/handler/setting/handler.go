// Package setting provides the HTTP handler for the settings menu page.
//
// [Ja] Package setting は設定メニューページの HTTP ハンドラーを提供します。
package setting

import (
	"github.com/mewstcom/mewst/go/internal/config"
)

// Handler is the HTTP handler for the settings menu page.
//
// [Ja] Handler は設定メニューページの HTTP ハンドラー。
type Handler struct {
	cfg *config.Config
}

// NewHandler creates a new Handler. The menu is a navigation hub with no
// persistence, so cfg (for page metadata) covers everything it renders.
//
// [Ja] NewHandler は新しい Handler を作成する。メニューは永続化を伴わない
// ナビゲーションハブのため、描画する内容はページメタデータ用の cfg で足りる。
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg: cfg,
	}
}
