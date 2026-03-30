// Package accounts はアカウント作成ハンドラーを提供します
package accounts

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// Handler はアカウント作成機能のHTTPハンドラー
type Handler struct {
	cfg                             *config.Config
	sessionMgr                      *session.Manager
	getSucceededEmailConfirmationUC *usecase.GetSucceededEmailConfirmationUsecase
	createAccountUC                 *usecase.CreateAccountUsecase
	createSessionUC                 *usecase.CreateSessionUsecase
	turnstile                       turnstile.Verifier
	rateLimiter                     *ratelimit.Limiter
	validator                       *validator.AccountsCreateValidator
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	getSucceededEmailConfirmationUC *usecase.GetSucceededEmailConfirmationUsecase,
	createAccountUC *usecase.CreateAccountUsecase,
	createSessionUC *usecase.CreateSessionUsecase,
	turnstile turnstile.Verifier,
	rateLimiter *ratelimit.Limiter,
	accountsValidator *validator.AccountsCreateValidator,
) *Handler {
	return &Handler{
		cfg:                             cfg,
		sessionMgr:                      sessionMgr,
		getSucceededEmailConfirmationUC: getSucceededEmailConfirmationUC,
		createAccountUC:                 createAccountUC,
		createSessionUC:                 createSessionUC,
		turnstile:                       turnstile,
		rateLimiter:                     rateLimiter,
		validator:                       accountsValidator,
	}
}
