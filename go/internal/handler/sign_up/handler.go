// Package sign_up はサインアップハンドラーを提供します
package sign_up

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// Handler はサインアップ機能のHTTPハンドラー
type Handler struct {
	cfg                       *config.Config
	sessionMgr                *session.Manager
	createEmailConfirmationUC *usecase.CreateEmailConfirmationUsecase
	turnstile                 turnstile.Verifier
	rateLimiter               *ratelimit.Limiter
	validator                 *validator.SignUpCreateValidator
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	createEmailConfirmationUC *usecase.CreateEmailConfirmationUsecase,
	turnstile turnstile.Verifier,
	rateLimiter *ratelimit.Limiter,
	signUpValidator *validator.SignUpCreateValidator,
) *Handler {
	return &Handler{
		cfg:                       cfg,
		sessionMgr:                sessionMgr,
		createEmailConfirmationUC: createEmailConfirmationUC,
		turnstile:                 turnstile,
		rateLimiter:               rateLimiter,
		validator:                 signUpValidator,
	}
}
