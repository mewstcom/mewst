// Package sign_out はログアウトハンドラーを提供します
package sign_out

import (
	"github.com/mewstcom/mewst/internal/config"
	"github.com/mewstcom/mewst/internal/session"
)

// Handler はログアウト機能のHTTPハンドラー
type Handler struct {
	cfg        *config.Config
	sessionMgr *session.Manager
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
) *Handler {
	return &Handler{
		cfg:        cfg,
		sessionMgr: sessionMgr,
	}
}
