package sign_up

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	sign_up_page "github.com/mewstcom/mewst/go/internal/templates/pages/sign_up"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// New はサインアップフォームを表示する (GET /sign_up)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールを設定（テンプレート内での翻訳用）
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページデータを作成
	data := sign_up_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       session.NewFormErrors(),
		Email:            "",
	}

	// メタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_up_title")
	meta.SetOGURL(h.cfg, r.URL.Path)

	// テンプレートをレンダリング
	content := sign_up_page.New(data)
	layout := layouts.Simple(meta, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
