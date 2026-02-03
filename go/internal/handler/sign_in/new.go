package sign_in

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	sign_in_page "github.com/mewstcom/mewst/go/internal/templates/pages/sign_in"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// New はログインフォームを表示する (GET /sign_in)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールと設定を設定（テンプレート内での翻訳用）
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// backパラメータを取得（ログイン後のリダイレクト先）
	backURL := r.URL.Query().Get("back")

	// ページデータを作成
	data := sign_in_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       session.NewFormErrors(),
		Email:            "",
		BackURL:          backURL,
	}

	// テンプレートをレンダリング
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_title")
	meta.SetOGURL(h.cfg, r.URL.Path)

	content := sign_in_page.New(data)
	layout := layouts.Simple(meta, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
