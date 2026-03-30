// Package handler はHTTPハンドラーの共通機能を提供します
package handler

import (
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/pages/errors"
)

// NotFound はスタイル付きの404ページをレンダリングする
func NotFound(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ロケールを設定
	locale := i18n.DetectLanguage(r)
	ctx = templates.WithLocale(ctx, locale)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	if err := errors.NotFound().Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "404ページのレンダリングに失敗", "error", err)
	}
}

// BadGateway はスタイル付きの502ページをレンダリングする
func BadGateway(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ロケールを設定
	locale := i18n.DetectLanguage(r)
	ctx = templates.WithLocale(ctx, locale)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)

	if err := errors.BadGateway().Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "502ページのレンダリングに失敗", "error", err)
	}
}
