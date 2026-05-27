// Package post provides HTTP handlers for posts.
// [Ja] Package post は投稿関連の HTTP ハンドラーを提供します。
package post

import (
	"github.com/mewstcom/mewst/go/internal/config"
)

// Handler is the HTTP handler for post-related endpoints.
// [Ja] Handler は投稿関連の HTTP ハンドラー。
type Handler struct {
	cfg *config.Config
}

// NewHandler creates a new Handler.
// [Ja] NewHandler は新しい Handler を作成する。
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg: cfg,
	}
}
