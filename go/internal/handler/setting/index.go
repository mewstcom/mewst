package setting

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	settingpages "github.com/mewstcom/mewst/go/internal/templates/pages/setting"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Index renders the settings menu (GET /settings). The page is a static
// navigation hub—links to the three subpages plus the sign-out form—so it needs
// no UseCase or DB access. The navbar profile and the CSRF token come from the
// context populated by RequireAuth and the CSRF middleware. The settings menu is
// not one of the navbar's five items, so the navbar renders with no active item.
//
// [Ja] Index は設定メニューを描画する (GET /settings)。このページは 3 つの
// サブページへのリンクとログアウトフォームだけの静的なナビゲーションハブのため、
// UseCase も DB アクセスも必要としない。navbar 用プロフィールと CSRF トークンは
// RequireAuth と CSRF ミドルウェアが context に格納したものを使う。設定メニューは
// navbar の 5 項目には含まれないため、navbar はアクティブ項目なしで描画する。
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "setting_index_title")

	profile := middleware.ProfileFromContext(ctx)
	navbar := viewmodel.NewNavbar(profile, viewmodel.NavbarItemNone)

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	content := settingpages.Index(settingpages.IndexPageData{CSRFToken: csrfToken})

	if err := layouts.Default(layouts.DefaultLayoutData{Meta: meta, Navbar: navbar}, content).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
