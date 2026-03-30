// Package email_confirmation はメール確認ハンドラーを提供します
package email_confirmation

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// Handler はメール確認機能のHTTPハンドラー
type Handler struct {
	cfg                          *config.Config
	sessionMgr                   *session.Manager
	getActiveEmailConfirmationUC *usecase.GetActiveEmailConfirmationUsecase
	markEmailAsConfirmedUC       *usecase.MarkEmailAsConfirmedUsecase
	validator                    *validator.EmailConfirmationCreateValidator
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	getActiveEmailConfirmationUC *usecase.GetActiveEmailConfirmationUsecase,
	markEmailAsConfirmedUC *usecase.MarkEmailAsConfirmedUsecase,
	ecValidator *validator.EmailConfirmationCreateValidator,
) *Handler {
	return &Handler{
		cfg:                          cfg,
		sessionMgr:                   sessionMgr,
		getActiveEmailConfirmationUC: getActiveEmailConfirmationUC,
		markEmailAsConfirmedUC:       markEmailAsConfirmedUC,
		validator:                    ecValidator,
	}
}
