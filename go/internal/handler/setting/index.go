package setting

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	settingpages "github.com/mewstcom/mewst/go/internal/templates/pages/setting"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Indexは設定メニューを描画する (GET /settings)。このページはサブページへの
// リンクとログアウトフォームだけのナビゲーションハブのため、自身で読み取るものは
// 無い。navbar用プロフィールとCSRFトークンはRequireAuthとCSRFミドルウェアが
// contextに格納したものを使う。設定メニューはnavbarの5項目には含まれないため、
// navbarはアクティブ項目なしで描画する。
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
