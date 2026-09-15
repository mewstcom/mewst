// Package email_confirmationはメール確認ハンドラーを提供します
package email_confirmation

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handlerはメール確認機能のHTTPハンドラー
type Handler struct {
	cfg                          *config.Config
	sessionMgr                   *session.Manager
	flashMgr                     *session.FlashManager
	getActiveEmailConfirmationUC *usecase.GetActiveEmailConfirmationUsecase
	verifyEmailConfirmationUC    *usecase.VerifyEmailConfirmationUsecase
}

// NewHandlerはHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	getActiveEmailConfirmationUC *usecase.GetActiveEmailConfirmationUsecase,
	verifyEmailConfirmationUC *usecase.VerifyEmailConfirmationUsecase,
) *Handler {
	return &Handler{
		cfg:                          cfg,
		sessionMgr:                   sessionMgr,
		flashMgr:                     flashMgr,
		getActiveEmailConfirmationUC: getActiveEmailConfirmationUC,
		verifyEmailConfirmationUC:    verifyEmailConfirmationUC,
	}
}
