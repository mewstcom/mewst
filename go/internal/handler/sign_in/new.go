package sign_in

import (
	"net/http"

	"github.com/mewstcom/mewst/internal/middleware"
	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
	"github.com/mewstcom/mewst/internal/templates/layouts"
	sign_in_page "github.com/mewstcom/mewst/internal/templates/pages/sign_in"
	"github.com/mewstcom/mewst/internal/viewmodel"
)

// New はログインフォームを表示する (GET /sign_in)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページデータを作成
	data := sign_in_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       session.NewFormErrors(),
		Email:            "",
	}

	// テンプレートをレンダリング
	meta := viewmodel.PageMeta{
		Title:        templates.T(ctx, "meta.title.sign_in.new"),
		Description:  "",
		AssetVersion: h.cfg.GetAssetVersion(),
	}

	// コンテキストにロケールを設定（テンプレート内での翻訳用）
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	content := sign_in_page.New(data)
	layout := layouts.Simple(meta, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
