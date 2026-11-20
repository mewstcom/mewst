// Package exportはエクスポートリソースのHTTPハンドラーを提供します。
package export

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// HandlerはエクスポートリソースのHTTPハンドラー。
type Handler struct {
	cfg             *config.Config
	flashMgr        *session.FlashManager
	getExportShowUC *usecase.GetExportShowUsecase
	createExportUC  *usecase.CreateExportUsecase
}

// NewHandlerは新しいHandlerを作成する。
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getExportShowUC *usecase.GetExportShowUsecase,
	createExportUC *usecase.CreateExportUsecase,
) *Handler {
	return &Handler{
		cfg:             cfg,
		flashMgr:        flashMgr,
		getExportShowUC: getExportShowUC,
		createExportUC:  createExportUC,
	}
}
