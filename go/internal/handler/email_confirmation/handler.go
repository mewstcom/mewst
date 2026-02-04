// Package email_confirmation はメール確認ハンドラーを提供します
package email_confirmation

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler はメール確認機能のHTTPハンドラー
type Handler struct {
	cfg                    *config.Config
	sessionMgr             *session.Manager
	emailConfirmationRepo  *repository.EmailConfirmationRepository
	markEmailAsConfirmedUC *usecase.MarkEmailAsConfirmedUsecase
	validator              *CreateValidator
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	markEmailAsConfirmedUC *usecase.MarkEmailAsConfirmedUsecase,
) *Handler {
	return &Handler{
		cfg:                    cfg,
		sessionMgr:             sessionMgr,
		emailConfirmationRepo:  emailConfirmationRepo,
		markEmailAsConfirmedUC: markEmailAsConfirmedUC,
		validator:              NewCreateValidator(emailConfirmationRepo),
	}
}
