package account

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
	accountpages "github.com/mewstcom/mewst/go/internal/templates/pages/account"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Create はアカウント作成処理を実行する (POST /accounts)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// クッキーからemail_confirmation_idを取得
	id, ok := h.sessionMgr.GetEmailConfirmationID(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// 確認済みのメール確認レコードを取得
	ecOutput, err := h.getSucceededEmailConfirmationUC.Execute(ctx, usecase.GetSucceededEmailConfirmationInput{ID: id})
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		slog.ErrorContext(ctx, "メール確認の取得に失敗", "error", err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// アカウント作成 / パスワード更新 / メール変更フローを取り違えてフォームに到達しないための防御。
	// アカウント作成は sign_up イベントのみ受け付ける。
	if ecOutput.EmailConfirmation.Event != model.EmailConfirmationEventSignUp {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// フォームデータを取得
	email := ecOutput.EmailConfirmation.Email
	atname := r.FormValue("atname")
	password := r.FormValue("password")
	backURL := r.FormValue("back")

	// IPアドレスベースのレート制限
	ipAddress := clientip.GetClientIP(r)
	if err := h.checkRateLimit(ctx, ipAddress); err != nil {
		if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
			ve := model.NewValidationError()
			ve.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
			h.renderAccountForm(w, r, ve, email, atname, backURL)
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
		h.renderAccountForm(w, r, ve, email, atname, backURL)
		return
	}

	// UseCase を実行（バリデーション + アカウント作成）
	accountOutput, err := h.createAccountUC.Execute(ctx, usecase.CreateAccountInput{
		Email:    email,
		Atname:   atname,
		Password: password,
		Locale:   i18n.GetLocale(ctx),
		TimeZone: "Asia/Tokyo",
	})
	if err != nil {
		h.handleCreateError(w, r, err, email, atname, backURL)
		return
	}

	// セッションを作成
	userAgent := r.UserAgent()
	sessionOutput, err := h.createSessionUC.Execute(ctx, usecase.CreateSessionInput{
		UserID:    accountOutput.Actor.UserID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
	if err != nil {
		slog.ErrorContext(ctx, "セッション作成中にエラーが発生", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションクッキーを設定
	h.sessionMgr.SetSessionCookie(w, sessionOutput.Token)

	// email_confirmation_idクッキーを削除
	h.sessionMgr.DeleteEmailConfirmationID(w)

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_account_created"))

	// /sign_in からの「アカウント登録」フローで戻り先を指定されていた場合はそこに戻す
	http.Redirect(w, r, redirect.GetSafeRedirectURL(backURL), http.StatusFound)
}

// checkRateLimit はIPアドレスベースのレート制限をチェックする
func (h *Handler) checkRateLimit(ctx context.Context, ipAddress string) error {
	return h.rateLimiter.Allow(ctx, ratelimit.CheckInput{
		Key:    ratelimit.IPKey(ipAddress),
		Limit:  5,
		Window: time.Minute,
	})
}

// handleCreateError はアカウント作成処理のエラーを処理する
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, email, atname, backURL string) {
	ctx := r.Context()

	var ve *model.ValidationError
	if errors.As(err, &ve) {
		h.renderAccountForm(w, r, ve, email, atname, backURL)
		return
	}

	slog.ErrorContext(ctx, "アカウント作成に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// renderAccountForm はアカウント作成フォームを再表示する
func (h *Handler) renderAccountForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, email, atname, backURL string) {
	ctx := r.Context()
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := accountpages.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       ve,
		Email:            email,
		Atname:           atname,
		BackURL:          backURL,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "account_new_title")

	content := accountpages.New(data)
	layout := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
