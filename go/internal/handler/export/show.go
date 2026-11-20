package export

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/httperror"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	exportpages "github.com/mewstcom/mewst/go/internal/templates/pages/export"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Showはエクスポート画面を描画する (GET /settings/export)。誰のエクスポートを
// 読むかはセッションのユーザーとプロフィールが決めるため、ハンドラーは両方をUseCase
// へ渡し、認可はUseCaseに委ねる。エクスポート画面はnavbarの5項目には含まれない
// ため、navbarはアクティブ項目なしで描画する。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// RequireAuthがこのハンドラーの前に両方をcontextへ格納するため、欠けて
	// いる場合は未ログインではなくRequireAuth無しでルートを登録したことを意味する。
	// 所有者が不明なままエクスポートを読まず、500で失敗させる。
	user := middleware.UserFromContext(ctx)
	profile := middleware.ProfileFromContext(ctx)
	if user == nil || profile == nil {
		slog.ErrorContext(ctx, "エクスポート画面でログイン中のユーザーまたはプロフィールを取得できませんでした")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	output, err := h.getExportShowUC.Execute(ctx, usecase.GetExportShowInput{
		UserID:    user.ID,
		ProfileID: profile.ID,
	})
	if err != nil {
		var ae *model.AppError
		if errors.As(err, &ae) {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound:
				httperror.NotFound(w, r)
			default:
				slog.ErrorContext(ctx, ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		slog.ErrorContext(ctx, "エクスポート画面の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "export_show_title")

	navbar := viewmodel.NewNavbar(profile, viewmodel.NavbarItemNone)

	content := exportpages.Show(exportpages.ShowPageData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Export: viewmodel.NewExport(viewmodel.ExportInput{
			Latest:          output.LatestExport,
			LatestSucceeded: output.LatestSucceededExport,
			Available:       output.Available,
		}),
	})

	if !output.Available {
		// templが最初の本文を書き込む前にWriteHeaderを呼ぶため、net/httpの
		// 自動判定に頼らずContent-Typeを明示する。
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := layouts.Default(layouts.DefaultLayoutData{Meta: meta, Navbar: navbar}, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "エクスポート画面のレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
