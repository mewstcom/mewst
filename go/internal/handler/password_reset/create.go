package password_reset

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	password_reset_page "github.com/mewstcom/mewst/go/internal/templates/pages/password_reset"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Create はパスワードリセット処理を実行する (POST /password_reset)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールを設定（テンプレート内での翻訳用）
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// フォームデータを取得
	email := r.FormValue("email")

	// Turnstileトークンを検証
	turnstileToken := r.FormValue("cf-turnstile-response")
	turnstileValid, err := h.turnstile.Verify(ctx, turnstileToken)
	if err != nil {
		slog.WarnContext(ctx, "Turnstile検証でエラーが発生", "error", err)
	}
	if !turnstileValid {
		ve := model.NewValidationError()
		ve.AddGlobal(templates.T(ctx, "error_turnstile_failed"))
		h.renderForm(w, r, ve, email)
		return
	}

	// UseCase を実行（バリデーション + メール確認レコード作成）
	ucResult, err := h.createPasswordResetUC.Execute(ctx, usecase.CreatePasswordResetInput{
		Email:  email,
		Locale: "ja",
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			h.renderForm(w, r, ve, email)
			return
		}

		slog.ErrorContext(ctx, "パスワードリセット処理に失敗", "error", err)
		// エラーが発生しても、セキュリティ上の理由でユーザーには成功メッセージを表示
		h.redirectToEmailConfirmation(w, r, ctx)
		return
	}

	// セッションにemail_confirmation_idを保存
	h.sessionMgr.SetEmailConfirmationID(w, r, ucResult.EmailConfirmation.ID.String())

	// /email_confirmationへリダイレクト
	h.redirectToEmailConfirmation(w, r, ctx)
}

// renderForm はパスワードリセットフォームを再表示する
func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, email string) {
	ctx := r.Context()
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := password_reset_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       ve,
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
func (h *Handler) redirectToEmailConfirmation(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "flash_password_reset_email_sent"))

	http.Redirect(w, r, "/email_confirmation", http.StatusFound)
}
