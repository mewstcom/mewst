// Package sign_outはログアウトハンドラーを提供します。
package sign_out

import (
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// HandlerはログアウトのHTTPハンドラー。
type Handler struct {
	sessionMgr      *session.Manager
	flashMgr        *session.FlashManager
	deleteSessionUC *usecase.DeleteSessionUsecase
}

// NewHandlerは新しいHandlerを作成する。
func NewHandler(
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	deleteSessionUC *usecase.DeleteSessionUsecase,
) *Handler {
	return &Handler{
		sessionMgr:      sessionMgr,
		flashMgr:        flashMgr,
		deleteSessionUC: deleteSessionUC,
	}
}
