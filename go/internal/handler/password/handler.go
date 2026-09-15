// Package passwordはパスワード関連のHTTPハンドラーを提供します
package password

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handlerはパスワード関連のHTTPハンドラー
type Handler struct {
	cfg                             *config.Config
	sessionMgr                      *session.Manager
	flashMgr                        *session.FlashManager
	getSucceededEmailConfirmationUC *usecase.GetSucceededEmailConfirmationUsecase
	updatePasswordUC                *usecase.UpdatePasswordUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	getSucceededEmailConfirmationUC *usecase.GetSucceededEmailConfirmationUsecase,
	updatePasswordUC *usecase.UpdatePasswordUsecase,
) *Handler {
	return &Handler{
		cfg:                             cfg,
		sessionMgr:                      sessionMgr,
		flashMgr:                        flashMgr,
		getSucceededEmailConfirmationUC: getSucceededEmailConfirmationUC,
		updatePasswordUC:                updatePasswordUC,
	}
}
