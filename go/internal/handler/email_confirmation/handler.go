// Package email_confirmation はメール確認ハンドラーを提供します
package email_confirmation

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler はメール確認機能のHTTPハンドラー
type Handler struct {
	cfg                          *config.Config
	sessionMgr                   *session.Manager
	getActiveEmailConfirmationUC *usecase.GetActiveEmailConfirmationUsecase
	verifyEmailConfirmationUC    *usecase.VerifyEmailConfirmationUsecase
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	getActiveEmailConfirmationUC *usecase.GetActiveEmailConfirmationUsecase,
	verifyEmailConfirmationUC *usecase.VerifyEmailConfirmationUsecase,
) *Handler {
	return &Handler{
		cfg:                          cfg,
		sessionMgr:                   sessionMgr,
		getActiveEmailConfirmationUC: getActiveEmailConfirmationUC,
		verifyEmailConfirmationUC:    verifyEmailConfirmationUC,
	}
}
