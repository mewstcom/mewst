// Package sign_in はログインハンドラーを提供します
package sign_in

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler はログイン機能のHTTPハンドラー
type Handler struct {
	cfg             *config.Config
	sessionMgr      *session.Manager
	actorRepo       *repository.ActorRepository
	createSessionUC *usecase.CreateSessionUsecase
	turnstile       turnstile.Verifier
	validator       *CreateValidator
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	userRepo *repository.UserRepository,
	actorRepo *repository.ActorRepository,
	createSessionUC *usecase.CreateSessionUsecase,
	turnstile turnstile.Verifier,
) *Handler {
	return &Handler{
		cfg:             cfg,
		sessionMgr:      sessionMgr,
		actorRepo:       actorRepo,
		createSessionUC: createSessionUC,
		turnstile:       turnstile,
		validator:       NewCreateValidator(userRepo),
	}
}
