package password_reset

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	password_reset_page "github.com/mewstcom/mewst/go/internal/templates/pages/password_reset"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// New はパスワードリセットフォームを表示する (GET /password_reset)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールと設定を設定（テンプレート内での翻訳用）
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページデータを作成
	data := password_reset_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       session.NewFormErrors(),
		Email:            "",
	}

	// テンプレートをレンダリング
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_reset_title")
	meta.SetOGURL(h.cfg, r.URL.Path)

	content := password_reset_page.New(data)
	layout := layouts.Simple(meta, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
