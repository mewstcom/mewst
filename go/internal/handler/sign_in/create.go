package sign_in

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/internal/auth"
	"github.com/mewstcom/mewst/internal/clientip"
	"github.com/mewstcom/mewst/internal/middleware"
	"github.com/mewstcom/mewst/internal/repository"
	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
	"github.com/mewstcom/mewst/internal/templates/layouts"
	sign_in_page "github.com/mewstcom/mewst/internal/templates/pages/sign_in"
	"github.com/mewstcom/mewst/internal/usecase"
)

// Create はログイン処理を実行する (POST /sign_in)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールを設定（テンプレート内での翻訳用）
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// フォームデータを取得
	req := &CreateRequest{
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
		formErrors.AddGlobalError(templates.T(ctx, "errors.turnstile_verification_failed"))
		h.renderForm(w, ctx, csrfToken, req.Email, formErrors)
		return
	}

	// フォームバリデーション
	formErrors := req.Validate(ctx)
	if formErrors.HasErrors() {
		h.renderForm(w, ctx, csrfToken, req.Email, formErrors)
		return
	}

	// ユーザーをメールアドレスで検索
	user, err := h.userRepo.GetByEmailForSignIn(ctx, req.Email)
	if err != nil {
		if err == repository.ErrNotFound {
			formErrors := session.NewFormErrors()
			formErrors.AddGlobalError(templates.T(ctx, "forms.errors.session_form.unauthenticated"))
			h.renderForm(w, ctx, csrfToken, req.Email, formErrors)
			return
		}
		slog.ErrorContext(ctx, "ユーザー検索中にエラーが発生", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// パスワードを検証
	if err := auth.CheckPassword(user.PasswordDigest, req.Password); err != nil {
		formErrors := session.NewFormErrors()
		formErrors.AddGlobalError(templates.T(ctx, "forms.errors.session_form.unauthenticated"))
		h.renderForm(w, ctx, csrfToken, req.Email, formErrors)
		return
	}

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
	result, err := h.createSessionUC.Execute(ctx, usecase.CreateSessionInput{
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
	h.sessionMgr.SetSessionCookie(w, r, result.Token)

	// フラッシュメッセージを設定
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "messages.authentication.sign_in"))

	// ホームページにリダイレクト
	http.Redirect(w, r, "/", http.StatusFound)
}

// renderForm はログインフォームを再表示する
func (h *Handler) renderForm(w http.ResponseWriter, ctx context.Context, csrfToken, email string, formErrors *session.FormErrors) {
	data := sign_in_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       formErrors,
		Email:            email,
	}

	meta := layouts.SimpleMeta{
		Title:        templates.T(ctx, "meta.title.sign_in.new"),
		Description:  "",
		AssetVersion: h.cfg.GetAssetVersion(),
	}

	content := sign_in_page.New(data)
	layout := layouts.Simple(meta, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
