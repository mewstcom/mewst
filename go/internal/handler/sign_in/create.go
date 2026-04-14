package sign_in

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/mewstcom/mewst/go/internal/clientip"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	sign_in_page "github.com/mewstcom/mewst/go/internal/templates/pages/sign_in"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Create はログイン処理を実行する (POST /sign_in)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールを設定（テンプレート内での翻訳用）
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// backパラメータを取得（ログイン後のリダイレクト先）
	backURL := r.FormValue("back")

	// フォームデータを取得
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Turnstileトークンを検証
	turnstileToken := r.FormValue("cf-turnstile-response")
	turnstileValid, err := h.turnstile.Verify(ctx, turnstileToken)
	if err != nil {
		slog.WarnContext(ctx, "Turnstile検証でエラーが発生", "error", err)
	}
	if !turnstileValid {
		ve := model.NewValidationError()
		ve.AddGlobal(templates.T(ctx, "error_turnstile_failed"))
		h.renderForm(w, r, ve, email, backURL)
		return
	}

	// UseCase を実行
	output, err := h.signInUC.Execute(ctx, usecase.CreateSignInInput{
		Email:     email,
		Password:  password,
		IPAddress: clientip.GetClientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		h.handleCreateError(w, r, err, email, backURL)
		return
	}

	// セッションクッキーを設定
	h.sessionMgr.SetSessionCookie(w, r, output.Token)

	// フラッシュメッセージを設定
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "flash_sign_in_success"))

	// リダイレクト先を決定（オープンリダイレクト攻撃を防ぐため相対パスのみ許可）
	redirectURL := "/"
	if backURL != "" && strings.HasPrefix(backURL, "/") && !strings.HasPrefix(backURL, "//") {
		redirectURL = backURL
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// handleCreateError はログイン処理のエラーを処理する
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, email string, backURL string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		h.renderForm(w, r, ve, email, backURL)
		return
	}

	slog.ErrorContext(ctx, "ログイン処理に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// renderForm はログインフォームを再表示する
func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, email, backURL string) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := sign_in_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       ve,
		Email:            email,
		BackURL:          backURL,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_title")
	meta.SetOGURL(h.cfg, "/sign_in")

	content := sign_in_page.New(data)
	layout := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
