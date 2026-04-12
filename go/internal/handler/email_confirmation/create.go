package email_confirmation

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	email_confirmation_page "github.com/mewstcom/mewst/go/internal/templates/pages/email_confirmation"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Create は確認コードを検証する (POST /email_confirmation)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールと設定を設定
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// クッキーからemail_confirmation_idを取得
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	if emailConfirmationID == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// UUIDをパース
	id, err := uuid.Parse(emailConfirmationID)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// フォームデータを取得
	code := r.FormValue("code")

	// UseCase を実行（バリデーション + 確認成功マーク）
	ucResult, err := h.verifyEmailConfirmationUC.Execute(ctx, usecase.VerifyEmailConfirmationInput{
		ID:   id,
		Code: code,
	})
	if err != nil {
		h.handleCreateError(w, r, err, code)
		return
	}

	// イベントに応じたリダイレクト先を決定
	redirectPath := h.getRedirectPath(ucResult.EmailConfirmation.Event)

	// フラッシュメッセージを設定
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "flash_email_confirmed"))

	http.Redirect(w, r, redirectPath, http.StatusFound)
}

// handleCreateError はメール確認処理のエラーを処理する
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, code string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		h.renderForm(w, r, ve, code)
		return
	}

	slog.ErrorContext(ctx, "メール確認処理に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// getRedirectPath はイベントに応じたリダイレクト先を返す
func (h *Handler) getRedirectPath(event model.EmailConfirmationEvent) string {
	switch event {
	case model.EmailConfirmationEventPasswordReset:
		return "/password/edit"
	case model.EmailConfirmationEventSignUp:
		return "/accounts/new"
	case model.EmailConfirmationEventEmailUpdate:
		return "/settings/email"
	default:
		return "/"
	}
}

// renderForm は確認コード入力フォームを再表示する
func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, code string) {
	ctx := r.Context()
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := email_confirmation_page.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: ve,
		Code:       code,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "email_confirmation_title")
	meta.SetOGURL(h.cfg, "/email_confirmation")

	content := email_confirmation_page.New(data)
	layout := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
