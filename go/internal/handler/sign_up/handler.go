// Package sign_upはサインアップハンドラーを提供します
package sign_up

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handlerはサインアップ機能のHTTPハンドラー
type Handler struct {
	cfg          *config.Config
	sessionMgr   *session.Manager
	flashMgr     *session.FlashManager
	createSignUp *usecase.CreateSignUpUsecase
	turnstile    turnstile.Verifier
	rateLimiter  *ratelimit.Limiter
}

// NewHandlerはHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	createSignUp *usecase.CreateSignUpUsecase,
	turnstile turnstile.Verifier,
	rateLimiter *ratelimit.Limiter,
) *Handler {
	return &Handler{
		cfg:          cfg,
		sessionMgr:   sessionMgr,
		flashMgr:     flashMgr,
		createSignUp: createSignUp,
		turnstile:    turnstile,
		rateLimiter:  rateLimiter,
	}
}
