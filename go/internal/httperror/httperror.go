// Package httperrorは全リソース共通のHTTPエラーレスポンスヘルパーを提供します。
// リソースディレクトリには属さない、404や502などの汎用エラーページのレンダリングを担当します。
package httperror

import (
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/templates/pages/errors"
)

// NotFoundはスタイル付きの404ページをレンダリングする
func NotFound(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	if err := errors.NotFound().Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "404ページのレンダリングに失敗", "error", err)
	}
}

// BadGatewayはスタイル付きの502ページをレンダリングする
func BadGateway(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)

	if err := errors.BadGateway().Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "502ページのレンダリングに失敗", "error", err)
	}
}
