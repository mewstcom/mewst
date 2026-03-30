// Package sign_in はログインハンドラーを提供します
package sign_in

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// Handler はログイン機能のHTTPハンドラー
type Handler struct {
	cfg             *config.Config
	sessionMgr      *session.Manager
	createSessionUC *usecase.CreateSessionUsecase
	turnstile       turnstile.Verifier
	validator       *validator.SignInCreateValidator
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	createSessionUC *usecase.CreateSessionUsecase,
	turnstile turnstile.Verifier,
	signInValidator *validator.SignInCreateValidator,
) *Handler {
	return &Handler{
		cfg:             cfg,
		sessionMgr:      sessionMgr,
		createSessionUC: createSessionUC,
		turnstile:       turnstile,
		validator:       signInValidator,
	}
}
