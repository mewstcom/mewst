// Package password はパスワード関連のHTTPハンドラーを提供します
package password

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// Handler はパスワード関連のHTTPハンドラー
type Handler struct {
	cfg                             *config.Config
	sessionMgr                      *session.Manager
	getSucceededEmailConfirmationUC *usecase.GetSucceededEmailConfirmationUsecase
	updatePasswordUC                *usecase.UpdatePasswordUsecase
	validator                       *validator.PasswordUpdateValidator
}

// NewHandler は新しいHandlerを作成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	getSucceededEmailConfirmationUC *usecase.GetSucceededEmailConfirmationUsecase,
	updatePasswordUC *usecase.UpdatePasswordUsecase,
	v *validator.PasswordUpdateValidator,
) *Handler {
	return &Handler{
		cfg:                             cfg,
		sessionMgr:                      sessionMgr,
		getSucceededEmailConfirmationUC: getSucceededEmailConfirmationUC,
		updatePasswordUC:                updatePasswordUC,
		validator:                       v,
	}
}
