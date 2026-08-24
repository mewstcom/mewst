package setting

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	settingpages "github.com/mewstcom/mewst/go/internal/templates/pages/setting"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Index renders the settings menu (GET /settings). The page is a navigation
// hub—links to the subpages plus the sign-out form—so the only thing it reads is
// the feature flag that decides whether the export entry is offered. The navbar
// profile and the CSRF token come from the context populated by RequireAuth and
// the CSRF middleware. The settings menu is not one of the navbar's five items,
// so the navbar renders with no active item.
//
// [Ja] Index は設定メニューを描画する (GET /settings)。このページはサブページへの
// リンクとログアウトフォームだけのナビゲーションハブのため、読み取るのはエクスポート
// 項目を提供するかを決めるフィーチャーフラグだけである。navbar 用プロフィールと CSRF
// トークンは RequireAuth と CSRF ミドルウェアが context に格納したものを使う。設定
// メニューは navbar の 5 項目には含まれないため、navbar はアクティブ項目なしで描画する。
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "setting_index_title")

	profile := middleware.ProfileFromContext(ctx)
	navbar := viewmodel.NewNavbar(profile, viewmodel.NavbarItemNone)

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	content := settingpages.Index(settingpages.IndexPageData{
		CSRFToken:     csrfToken,
		ExportEnabled: h.exportEnabled(ctx),
	})

	if err := layouts.Default(layouts.DefaultLayoutData{Meta: meta, Navbar: navbar}, content).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// exportEnabled reports whether the export entry belongs in this viewer's menu.
//
// Both failure paths hide the entry and let the rest of the menu render. The
// export page is still unreleased, so hiding it costs a viewer nothing they
// have today, while failing the request would take the profile, user, and email
// settings down with it. The reverse proxy gates the export page on the same
// flag and also falls back to the Rails version when its lookup errors, so both
// layers fail in the same direction.
//
// [Ja] exportEnabled は閲覧者のメニューにエクスポート項目を含めるかを返す。
//
// どちらの失敗経路でも項目を隠し、メニューの残りは描画する。エクスポート画面は
// まだ未公開のため、隠しても閲覧者が今日持っているものは失われないが、リクエストを
// 失敗させるとプロフィール・ユーザー・メールアドレスの設定まで巻き添えで落ちる。
// リバースプロキシもエクスポート画面を同じフラグで制御し、判定でエラーが起きたときは
// Rails 版へフォールバックするため、両者の失敗方向は揃う。
func (h *Handler) exportEnabled(ctx context.Context) bool {
	// RequireAuth puts the actor on the context before this handler runs, so a
	// missing one means the route was wired without it rather than a signed-out
	// visitor.
	//
	// [Ja] RequireAuth がこのハンドラーの前に actor を context へ格納するため、
	// 欠けている場合は未ログインではなく RequireAuth 無しでルートを登録したことを
	// 意味する。
	actor := middleware.ActorFromContext(ctx)
	if actor == nil {
		slog.ErrorContext(ctx, "設定メニューでログイン中のアクターを取得できませんでした")
		return false
	}

	output, err := h.getSettingIndexUC.Execute(ctx, usecase.GetSettingIndexInput{ActorID: actor.ID})
	if err != nil {
		slog.WarnContext(ctx, "設定メニューのエクスポート項目のフィーチャーフラグ判定に失敗", "error", err)
		return false
	}

	return output.ExportEnabled
}
