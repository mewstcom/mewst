// Package password_reset はパスワードリセット開始ハンドラーを提供します
package password_reset

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler はパスワードリセット開始機能のHTTPハンドラー
type Handler struct {
	cfg                   *config.Config
	sessionMgr            *session.Manager
	createPasswordResetUC *usecase.CreatePasswordResetUsecase
	turnstile             turnstile.Verifier
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	createPasswordResetUC *usecase.CreatePasswordResetUsecase,
	turnstile turnstile.Verifier,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		sessionMgr:            sessionMgr,
		createPasswordResetUC: createPasswordResetUC,
		turnstile:             turnstile,
	}
}
