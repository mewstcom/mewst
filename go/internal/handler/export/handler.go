// Package export provides HTTP handlers for the export resource.
//
// [Ja] Package export はエクスポートリソースの HTTP ハンドラーを提供します。
package export

import (
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler is the HTTP handler for the export resource.
//
// [Ja] Handler はエクスポートリソースの HTTP ハンドラー。
type Handler struct {
	cfg             *config.Config
	getExportShowUC *usecase.GetExportShowUsecase
}

// NewHandler creates a new Handler.
//
// [Ja] NewHandler は新しい Handler を作成する。
func NewHandler(
	cfg *config.Config,
	getExportShowUC *usecase.GetExportShowUsecase,
) *Handler {
	return &Handler{
		cfg:             cfg,
		getExportShowUC: getExportShowUC,
	}
}
