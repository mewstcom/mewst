// Package password_reset はパスワードリセット開始ハンドラーを提供します
package password_reset

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// Handler はパスワードリセット開始機能のHTTPハンドラー
type Handler struct {
	cfg                       *config.Config
	sessionMgr                *session.Manager
	createEmailConfirmationUC *usecase.CreateEmailConfirmationUsecase
	turnstile                 turnstile.Verifier
	validator                 *validator.PasswordResetCreateValidator
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	createEmailConfirmationUC *usecase.CreateEmailConfirmationUsecase,
	turnstile turnstile.Verifier,
	v *validator.PasswordResetCreateValidator,
) *Handler {
	return &Handler{
		cfg:                       cfg,
		sessionMgr:                sessionMgr,
		createEmailConfirmationUC: createEmailConfirmationUC,
		turnstile:                 turnstile,
		validator:                 v,
	}
}
