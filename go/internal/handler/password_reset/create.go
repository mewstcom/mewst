package password_reset

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/internal/middleware"
	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
	"github.com/mewstcom/mewst/internal/templates/layouts"
	password_reset_page "github.com/mewstcom/mewst/internal/templates/pages/password_reset"
	"github.com/mewstcom/mewst/internal/usecase"
	"github.com/mewstcom/mewst/internal/viewmodel"
)

// Create はパスワードリセット処理を実行する (POST /password_reset)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールを設定（テンプレート内での翻訳用）
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// フォームデータを取得
	req := &CreateRequest{
		Email: r.FormValue("email"),
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
		h.renderForm(w, ctx, csrfToken, req.Email, formErrors)
		return
	}

	// フォームバリデーション
	formErrors := req.Validate(ctx)
	if formErrors.HasErrors() {
		h.renderForm(w, ctx, csrfToken, req.Email, formErrors)
		return
	}

	// メール確認レコードを作成し、確認メールを送信
	result, err := h.createEmailConfirmationUC.Execute(ctx, usecase.CreateEmailConfirmationInput{
		Email:  req.Email,
		Event:  model.EmailConfirmationEventPasswordReset,
		Locale: "ja",
	})
	if err != nil {
		slog.ErrorContext(ctx, "メール確認レコードの作成に失敗", "error", err)
		// エラーが発生しても、セキュリティ上の理由でユーザーには成功メッセージを表示
		// （メールアドレスの存在確認を防ぐため）
		h.redirectToEmailConfirmation(w, r, ctx, "")
		return
	}

	// セッションにemail_confirmation_idを保存
	h.sessionMgr.SetEmailConfirmationID(w, r, result.EmailConfirmation.ID.String())

	// /email_confirmationへリダイレクト
	h.redirectToEmailConfirmation(w, r, ctx, result.EmailConfirmation.ID.String())
}

// renderForm はパスワードリセットフォームを再表示する
func (h *Handler) renderForm(w http.ResponseWriter, ctx context.Context, csrfToken, email string, formErrors *session.FormErrors) {
	data := password_reset_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       formErrors,
		Email:            email,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_reset_title")
	meta.SetOGURL(h.cfg, "/password_reset")

	content := password_reset_page.New(data)
	layout := layouts.Simple(meta, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// redirectToEmailConfirmation はメール確認ページへリダイレクトする
func (h *Handler) redirectToEmailConfirmation(w http.ResponseWriter, r *http.Request, ctx context.Context, _ string) {
	// フラッシュメッセージを設定
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "flash_password_reset_email_sent"))

	http.Redirect(w, r, "/email_confirmation", http.StatusFound)
}
