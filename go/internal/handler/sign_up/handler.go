// Package sign_up はサインアップハンドラーを提供します
package sign_up

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler はサインアップ機能のHTTPハンドラー
type Handler struct {
	cfg                       *config.Config
	sessionMgr                *session.Manager
	userRepo                  *repository.UserRepository
	createEmailConfirmationUC *usecase.CreateEmailConfirmationUsecase
	turnstile                 turnstile.Verifier
	rateLimiter               *ratelimit.Limiter
	validator                 *CreateValidator
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	userRepo *repository.UserRepository,
	createEmailConfirmationUC *usecase.CreateEmailConfirmationUsecase,
	turnstile turnstile.Verifier,
	rateLimiter *ratelimit.Limiter,
) *Handler {
	return &Handler{
		cfg:                       cfg,
		sessionMgr:                sessionMgr,
		userRepo:                  userRepo,
		createEmailConfirmationUC: createEmailConfirmationUC,
		turnstile:                 turnstile,
		rateLimiter:               rateLimiter,
		validator:                 NewCreateValidator(userRepo),
	}
}
