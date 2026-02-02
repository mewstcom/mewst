// Package email_confirmation はメール確認ハンドラーを提供します
package email_confirmation

import (
	"github.com/mewstcom/mewst/internal/config"
	"github.com/mewstcom/mewst/internal/repository"
	"github.com/mewstcom/mewst/internal/session"
)

// Handler はメール確認機能のHTTPハンドラー
type Handler struct {
	cfg                   *config.Config
	sessionMgr            *session.Manager
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		sessionMgr:            sessionMgr,
		emailConfirmationRepo: emailConfirmationRepo,
	}
}
