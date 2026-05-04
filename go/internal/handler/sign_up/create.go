package sign_up

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/mewstcom/mewst/go/internal/clientip"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/redirect"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	signuppages "github.com/mewstcom/mewst/go/internal/templates/pages/sign_up"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Create はサインアップ処理を実行する (POST /sign_up)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// フォームデータを取得
	email := r.FormValue("email")
	backURL := r.FormValue("back")

	// IPアドレスベースのレート制限
	ipAddress := clientip.GetClientIP(r)
	if err := h.checkRateLimit(ctx, ipAddress); err != nil {
		if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
			ve := model.NewValidationError()
			ve.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
			h.renderSignUpForm(w, r, ve, email, backURL)
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
		ve := model.NewValidationError()
		ve.AddGlobal(i18n.T(ctx, "validation_turnstile_failed"))
		h.renderSignUpForm(w, r, ve, email, backURL)
		return
	}

	// UseCase を実行（バリデーション + メール確認レコード作成）
	ucResult, err := h.createSignUp.Execute(ctx, usecase.CreateSignUpInput{
		Email:  email,
		Locale: i18n.GetLocale(ctx),
	})
	if err != nil {
		h.handleCreateError(w, r, err, email, backURL)
		return
	}

	// セッションにemail_confirmation_idを保存
	h.sessionMgr.SetEmailConfirmationID(w, ucResult.EmailConfirmation.ID)

	// フラッシュメッセージを設定して/email_confirmationへリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_sign_up_email_sent"))

	http.Redirect(w, r, redirect.AppendSafeBack("/email_confirmation", backURL), http.StatusFound)
}

// handleCreateError はサインアップ処理のエラーを処理する
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, email, backURL string) {
	ctx := r.Context()

	var ve *model.ValidationError
	if errors.As(err, &ve) {
		h.renderSignUpForm(w, r, ve, email, backURL)
		return
	}

	slog.ErrorContext(ctx, "サインアップ処理に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// checkRateLimit はIPアドレスベースのレート制限をチェックする
func (h *Handler) checkRateLimit(ctx context.Context, ipAddress string) error {
	return h.rateLimiter.Allow(ctx, ratelimit.CheckInput{
		Key:    ratelimit.IPKey(ipAddress),
		Limit:  5,
		Window: time.Minute,
	})
}

// renderSignUpForm はサインアップフォームを再表示する
func (h *Handler) renderSignUpForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, email, backURL string) {
	ctx := r.Context()
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := signuppages.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       ve,
		Email:            email,
		BackURL:          backURL,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_up_new_title")

	content := signuppages.New(data)
	layout := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
