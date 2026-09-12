// Package export_download provides the HTTP handler that hands over an
// export archive. It is a resource of its own rather than an action on the
// export resource, because the export page and the export start render and
// redirect HTML while this one streams a file: the two share no response
// shape, and the handler naming convention has no file name for a custom
// action.
//
// [Ja] Package export_download はエクスポートのアーカイブを渡す HTTP ハンドラーを
// 提供します。エクスポートリソースのアクションではなく独立したリソースとしているのは、
// エクスポート画面と開始が HTML を描画・リダイレクトするのに対し、こちらはファイルを
// ストリーミングするためです。両者はレスポンスの形を共有せず、ハンドラーの命名規約も
// カスタムアクションのファイル名を持ちません。
package export_download

import (
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Handler is the HTTP handler for downloading an export.
//
// [Ja] Handler はエクスポートのダウンロードの HTTP ハンドラー。
type Handler struct {
	getExportDownloadUC *usecase.GetExportDownloadUsecase
}

// NewHandler creates a new Handler.
//
// [Ja] NewHandler は新しい Handler を作成する。
func NewHandler(getExportDownloadUC *usecase.GetExportDownloadUsecase) *Handler {
	return &Handler{
		getExportDownloadUC: getExportDownloadUC,
	}
}
