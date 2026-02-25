package accounts

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/mewstcom/mewst/go/internal/clientip"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	accounts_page "github.com/mewstcom/mewst/go/internal/templates/pages/accounts"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Create はアカウント作成処理を実行する (POST /accounts)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// セッションからemail_confirmation_idを取得
	emailConfirmation, err := h.getVerifiedEmailConfirmation(r)
	if err != nil {
		slog.ErrorContext(ctx, "メール確認の取得に失敗", "error", err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if emailConfirmation == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// フォームデータを取得
	input := CreateValidatorInput{
		Email:    emailConfirmation.Email,
		Atname:   r.FormValue("atname"),
		Password: r.FormValue("password"),
	}

	// IPアドレスベースのレート制限
	ipAddress := clientip.GetClientIP(r)
	if err := h.checkRateLimit(ctx, ipAddress); err != nil {
		if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
			formErrors := session.NewFormErrors()
			formErrors.AddGlobalError(templates.T(ctx, "error_rate_limit_exceeded"))
			h.renderForm(w, ctx, csrfToken, input.Email, input.Atname, formErrors)
			return
		}
		slog.ErrorContext(ctx, "レート制限チェックでエラー", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
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
		h.renderForm(w, ctx, csrfToken, input.Email, input.Atname, formErrors)
		return
	}

	// フォームバリデーション
	result := h.validator.Validate(ctx, input)
	if result.Err != nil {
		slog.ErrorContext(ctx, "バリデーション中にエラー", "error", result.Err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if result.FormErrors.HasErrors() {
		h.renderForm(w, ctx, csrfToken, input.Email, input.Atname, result.FormErrors)
		return
	}

	// アカウントを作成
	accountResult, err := h.createAccountUC.Execute(ctx, usecase.CreateAccountInput{
		Email:    input.Email,
		Atname:   input.Atname,
		Password: input.Password,
		Locale:   "ja",
		TimeZone: "Asia/Tokyo",
	})
	if err != nil {
		slog.ErrorContext(ctx, "アカウントの作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションを作成
	userAgent := r.UserAgent()
	sessionResult, err := h.createSessionUC.Execute(ctx, usecase.CreateSessionInput{
		ActorID:   accountResult.Actor.ID,
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

	// email_confirmation_idクッキーを削除
	h.sessionMgr.DeleteEmailConfirmationID(w, r)

	// フラッシュメッセージを設定
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "flash_account_created"))

	http.Redirect(w, r, "/", http.StatusFound)
}

// checkRateLimit はIPアドレスベースのレート制限をチェックする
func (h *Handler) checkRateLimit(ctx context.Context, ipAddress string) error {
	return h.rateLimiter.Allow(ctx, ratelimit.CheckInput{
		Key:    ratelimit.IPKey(ipAddress),
		Limit:  5,
		Window: time.Minute,
	})
}

// renderForm はアカウント作成フォームを再表示する
func (h *Handler) renderForm(w http.ResponseWriter, ctx context.Context, csrfToken, email, atname string, formErrors *session.FormErrors) {
	data := accounts_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       formErrors,
		Email:            email,
		Atname:           atname,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "accounts_new_title")
	meta.SetOGURL(h.cfg, "/accounts/new")

	content := accounts_page.New(data)
	layout := layouts.Simple(meta, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
