package email_confirmation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/internal/middleware"
	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/repository"
	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
	"github.com/mewstcom/mewst/internal/templates/layouts"
	email_confirmation_page "github.com/mewstcom/mewst/internal/templates/pages/email_confirmation"
	"github.com/mewstcom/mewst/internal/viewmodel"
)

// Create は確認コードを検証する (POST /email_confirmation)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールと設定を設定
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// クッキーからemail_confirmation_idを取得
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	if emailConfirmationID == "" {
		// IDがない場合はルートにリダイレクト
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// UUIDをパース
	id, err := uuid.Parse(emailConfirmationID)
	if err != nil {
		// 無効なIDの場合はルートにリダイレクト
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// フォームデータを取得
	req := &CreateRequest{
		Code: r.FormValue("code"),
	}

	// フォームバリデーション
	formErrors := req.Validate(ctx)
	if formErrors.HasErrors() {
		h.renderForm(w, ctx, csrfToken, req.Code, formErrors)
		return
	}

	// 有効な確認レコードを取得
	emailConfirmation, err := h.emailConfirmationRepo.GetActiveByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// 見つからないまたは期限切れの場合はエラー表示
			formErrors := session.NewFormErrors()
			formErrors.AddGlobalError(templates.T(ctx, "error_code_incorrect_or_expired"))
			h.renderForm(w, ctx, csrfToken, req.Code, formErrors)
			return
		}
		slog.ErrorContext(ctx, "メール確認レコードの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 確認コードを検証
	if emailConfirmation.Code != req.Code {
		formErrors := session.NewFormErrors()
		formErrors.AddGlobalError(templates.T(ctx, "error_code_incorrect_or_expired"))
		h.renderForm(w, ctx, csrfToken, req.Code, formErrors)
		return
	}

	// 確認を成功としてマーク
	if err := h.emailConfirmationRepo.MarkAsSucceeded(ctx, id); err != nil {
		slog.ErrorContext(ctx, "メール確認の成功マークに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// イベントに応じたリダイレクト先を決定
	redirectPath := h.getRedirectPath(emailConfirmation.Event)

	// フラッシュメッセージを設定
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "flash_email_confirmed"))

	http.Redirect(w, r, redirectPath, http.StatusFound)
}

// getRedirectPath はイベントに応じたリダイレクト先を返す
func (h *Handler) getRedirectPath(event model.EmailConfirmationEvent) string {
	switch event {
	case model.EmailConfirmationEventPasswordReset:
		return "/password/edit"
	case model.EmailConfirmationEventSignUp:
		return "/sign_up/new_account"
	case model.EmailConfirmationEventEmailUpdate:
		return "/settings/email"
	default:
		return "/"
	}
}

// renderForm は確認コード入力フォームを再表示する
func (h *Handler) renderForm(w http.ResponseWriter, ctx context.Context, csrfToken, code string, formErrors *session.FormErrors) {
	data := email_confirmation_page.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
		Code:       code,
		Flash:      nil,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "email_confirmation_title")
	meta.SetOGURL(h.cfg, "/email_confirmation")

	content := email_confirmation_page.New(data)
	layout := layouts.Simple(meta, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
