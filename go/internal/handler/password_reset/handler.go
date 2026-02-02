// Package password_reset はパスワードリセット開始ハンドラーを提供します
package password_reset

import (
	"github.com/mewstcom/mewst/internal/config"
	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/turnstile"
	"github.com/mewstcom/mewst/internal/usecase"
)

// Handler はパスワードリセット開始機能のHTTPハンドラー
type Handler struct {
	cfg                       *config.Config
	sessionMgr                *session.Manager
	createEmailConfirmationUC *usecase.CreateEmailConfirmationUsecase
	turnstile                 turnstile.Verifier
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	createEmailConfirmationUC *usecase.CreateEmailConfirmationUsecase,
	turnstile turnstile.Verifier,
) *Handler {
	return &Handler{
		cfg:                       cfg,
		sessionMgr:                sessionMgr,
		createEmailConfirmationUC: createEmailConfirmationUC,
		turnstile:                 turnstile,
	}
}
