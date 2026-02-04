package sign_in

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mewstcom/mewst/go/internal/clientip"
	"github.com/mewstcom/mewst/go/internal/middleware"
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

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// backパラメータを取得（ログイン後のリダイレクト先）
	backURL := r.FormValue("back")

	// フォームデータを取得
	input := CreateValidatorInput{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	// Turnstileトークンを検証
	turnstileToken := r.FormValue("cf-turnstile-response")
	turnstileValid, err := h.turnstile.Verify(ctx, turnstileToken)
	if err != nil {
		slog.WarnContext(ctx, "Turnstile検証でエラーが発生", "error", err)
	}
	if !turnstileValid {
		formErrors := session.NewFormErrors()
		formErrors.AddGlobalError(templates.T(ctx, "error_turnstile_failed"))
		h.renderForm(w, ctx, csrfToken, input.Email, backURL, formErrors)
		return
	}

	// バリデーション（形式バリデーション + 状態バリデーション）
	result := h.validator.Validate(ctx, input)
	if result.Err != nil {
		slog.ErrorContext(ctx, "ユーザー検索中にエラーが発生", "error", result.Err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if result.FormErrors != nil && result.FormErrors.HasErrors() {
		h.renderForm(w, ctx, csrfToken, input.Email, backURL, result.FormErrors)
		return
	}

	user := result.User

	// アクターを取得
	actor, err := h.actorRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		slog.ErrorContext(ctx, "アクター取得中にエラーが発生", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// IPアドレスとUser-Agentを取得
	ipAddress := clientip.GetClientIP(r)
	userAgent := r.UserAgent()

	// セッションを作成
	sessionResult, err := h.createSessionUC.Execute(ctx, usecase.CreateSessionInput{
		ActorID:   actor.ID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
	if err != nil {
		slog.ErrorContext(ctx, "セッション作成中にエラーが発生", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションクッキーを設定
	h.sessionMgr.SetSessionCookie(w, r, sessionResult.Token)

	// フラッシュメッセージを設定
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "flash_sign_in_success"))

	// リダイレクト先を決定（オープンリダイレクト攻撃を防ぐため相対パスのみ許可）
	redirectURL := "/"
	if backURL != "" && strings.HasPrefix(backURL, "/") && !strings.HasPrefix(backURL, "//") {
		redirectURL = backURL
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// renderForm はログインフォームを再表示する
func (h *Handler) renderForm(w http.ResponseWriter, ctx context.Context, csrfToken, email, backURL string, formErrors *session.FormErrors) {
	data := sign_in_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       formErrors,
		Email:            email,
		BackURL:          backURL,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_title")
	meta.SetOGURL(h.cfg, "/sign_in")

	content := sign_in_page.New(data)
	layout := layouts.Simple(meta, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
