package setting

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	settingpages "github.com/mewstcom/mewst/go/internal/templates/pages/setting"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Index renders the settings menu (GET /settings). The page is a navigation
// hub—links to the subpages plus the sign-out form—so it reads nothing of its
// own. The navbar profile and the CSRF token come from the context populated by
// RequireAuth and the CSRF middleware. The settings menu is not one of the
// navbar's five items, so the navbar renders with no active item.
//
// [Ja] Index は設定メニューを描画する (GET /settings)。このページはサブページへの
// リンクとログアウトフォームだけのナビゲーションハブのため、自身で読み取るものは
// 無い。navbar 用プロフィールと CSRF トークンは RequireAuth と CSRF ミドルウェアが
// context に格納したものを使う。設定メニューは navbar の 5 項目には含まれないため、
// navbar はアクティブ項目なしで描画する。
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "setting_index_title")

	profile := middleware.ProfileFromContext(ctx)
	navbar := viewmodel.NewNavbar(profile, viewmodel.NavbarItemNone)

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	content := settingpages.Index(settingpages.IndexPageData{
		CSRFToken: csrfToken,
	})

	if err := layouts.Default(layouts.DefaultLayoutData{Meta: meta, Navbar: navbar}, content).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
