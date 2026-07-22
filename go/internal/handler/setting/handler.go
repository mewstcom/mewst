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

// NewHandler creates a new Handler. The settings menu is a static navigation
// hub with no persistence, so cfg (for page metadata) is the only dependency;
// no UseCase is needed.
//
// [Ja] NewHandler は新しい Handler を作成する。設定メニューは永続化を伴わない
// 静的なナビゲーションハブのため、依存はページメタデータ用の cfg のみで、UseCase は
// 不要。
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg: cfg,
	}
}
