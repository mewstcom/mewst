package sign_up

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/mewstcom/mewst/go/internal/clientip"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	sign_up_page "github.com/mewstcom/mewst/go/internal/templates/pages/sign_up"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Create はサインアップ処理を実行する (POST /sign_up)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールを設定（テンプレート内での翻訳用）
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// フォームデータを取得
	input := validator.SignUpCreateValidatorInput{
		Email: r.FormValue("email"),
	}

	// IPアドレスベースのレート制限
	ipAddress := clientip.GetClientIP(r)
	if err := h.checkRateLimit(ctx, ipAddress); err != nil {
		if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
			formErrors := session.NewFormErrors()
			formErrors.AddGlobalError(templates.T(ctx, "error_rate_limit_exceeded"))
			h.renderForm(w, ctx, csrfToken, input.Email, formErrors)
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
		h.renderForm(w, ctx, csrfToken, input.Email, formErrors)
		return
	}

	// フォームバリデーション
	result := h.validator.Validate(ctx, input)
	if result.Err != nil {
		slog.ErrorContext(ctx, "バリデーション中にエラー", "error", result.Err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if result.FormErrors != nil && result.FormErrors.HasErrors() {
		h.renderForm(w, ctx, csrfToken, input.Email, result.FormErrors)
		return
	}

	// メール確認レコードを作成し、確認メールを送信
	ucResult, err := h.createEmailConfirmationUC.Execute(ctx, usecase.CreateEmailConfirmationInput{
		Email:  input.Email,
		Event:  model.EmailConfirmationEventSignUp,
		Locale: "ja",
	})
	if err != nil {
		slog.ErrorContext(ctx, "メール確認レコードの作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションにemail_confirmation_idを保存
	h.sessionMgr.SetEmailConfirmationID(w, r, ucResult.EmailConfirmation.ID.String())

	// フラッシュメッセージを設定して/email_confirmationへリダイレクト
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "flash_sign_up_email_sent"))

	http.Redirect(w, r, "/email_confirmation", http.StatusFound)
}

// checkRateLimit はIPアドレスベースのレート制限をチェックする
func (h *Handler) checkRateLimit(ctx context.Context, ipAddress string) error {
	return h.rateLimiter.Allow(ctx, ratelimit.CheckInput{
		Key:    ratelimit.IPKey(ipAddress),
		Limit:  5,
		Window: time.Minute,
	})
}

// renderForm はサインアップフォームを再表示する
func (h *Handler) renderForm(w http.ResponseWriter, ctx context.Context, csrfToken, email string, formErrors *session.FormErrors) {
	data := sign_up_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       formErrors,
		Email:            email,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_up_title")
	meta.SetOGURL(h.cfg, "/sign_up")

	content := sign_up_page.New(data)
	layout := layouts.Simple(meta, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
