// Package sign_out provides the sign-out handler.
//
// [Ja] Package sign_out はログアウトハンドラーを提供します。
package sign_out

import (
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler is the HTTP handler for the sign-out endpoint.
//
// [Ja] Handler はログアウトの HTTP ハンドラー。
type Handler struct {
	sessionMgr      *session.Manager
	flashMgr        *session.FlashManager
	deleteSessionUC *usecase.DeleteSessionUsecase
}

// NewHandler creates a new Handler.
//
// [Ja] NewHandler は新しい Handler を作成する。
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
