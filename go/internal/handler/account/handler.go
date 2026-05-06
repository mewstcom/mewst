// Package account はアカウント作成ハンドラーを提供します
package account

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler はアカウント作成機能のHTTPハンドラー
type Handler struct {
	cfg                             *config.Config
	sessionMgr                      *session.Manager
	flashMgr                        *session.FlashManager
	getSucceededEmailConfirmationUC *usecase.GetSucceededEmailConfirmationUsecase
	createAccountUC                 *usecase.CreateAccountUsecase
	createSessionUC                 *usecase.CreateSessionUsecase
	turnstile                       turnstile.Verifier
	rateLimiter                     *ratelimit.Limiter
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	getSucceededEmailConfirmationUC *usecase.GetSucceededEmailConfirmationUsecase,
	createAccountUC *usecase.CreateAccountUsecase,
	createSessionUC *usecase.CreateSessionUsecase,
	turnstile turnstile.Verifier,
	rateLimiter *ratelimit.Limiter,
) *Handler {
	return &Handler{
		cfg:                             cfg,
		sessionMgr:                      sessionMgr,
		flashMgr:                        flashMgr,
		getSucceededEmailConfirmationUC: getSucceededEmailConfirmationUC,
		createAccountUC:                 createAccountUC,
		createSessionUC:                 createSessionUC,
		turnstile:                       turnstile,
		rateLimiter:                     rateLimiter,
	}
}
