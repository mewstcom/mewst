// Package export_downloadはエクスポートのアーカイブを渡すHTTPハンドラーを
// 提供します。エクスポートリソースのアクションではなく独立したリソースとしているのは、
// エクスポート画面と開始がHTMLを描画・リダイレクトするのに対し、こちらはファイルを
// ストリーミングするためです。両者はレスポンスの形を共有せず、ハンドラーの命名規約も
// カスタムアクションのファイル名を持ちません。
package export_download

import (
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// HandlerはエクスポートのダウンロードのHTTPハンドラー。
type Handler struct {
	getExportDownloadUC *usecase.GetExportDownloadUsecase
}

// NewHandlerは新しいHandlerを作成する。
func NewHandler(getExportDownloadUC *usecase.GetExportDownloadUsecase) *Handler {
	return &Handler{
		getExportDownloadUC: getExportDownloadUC,
	}
}
