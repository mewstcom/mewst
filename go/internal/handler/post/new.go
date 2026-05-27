package post

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	postpages "github.com/mewstcom/mewst/go/internal/templates/pages/post"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// New renders the new post form (GET /new). The current profile and CSRF token
// come from the context populated by RequireAuth and the CSRF middleware, so
// the handler only needs to assemble the layout and render.
//
// [Ja] New は新規投稿フォームを表示する (GET /new)。現在プロフィールと CSRF
// トークンは RequireAuth と CSRF ミドルウェアが context に格納するため、
// ハンドラーはレイアウトを組み立てて描画するだけでよい。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)
	profile := middleware.ProfileFromContext(ctx)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "post_new_title")

	data := layouts.DefaultLayoutData{
		Meta:   meta,
		Navbar: viewmodel.NewNavbar(profile, viewmodel.NavbarItemNew),
	}
	content := postpages.New(postpages.NewPageData{
		CSRFToken: csrfToken,
	})

	if err := layouts.Default(data, content).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
