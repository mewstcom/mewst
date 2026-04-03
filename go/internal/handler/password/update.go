package password

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	password_page "github.com/mewstcom/mewst/go/internal/templates/pages/password"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Update はパスワードを更新する (PATCH /password)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	// 確認済みのメール確認レコードを取得
	ecResult, err := h.getSucceededEmailConfirmationUC.Execute(ctx, usecase.GetSucceededEmailConfirmationInput{ID: id})
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		slog.ErrorContext(ctx, "メール確認レコードの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// UseCase を実行（バリデーション + パスワード更新）
	password := r.FormValue("password")
	ucInput := usecase.UpdatePasswordInput{
		Email:    ecResult.EmailConfirmation.Email,
		Password: password,
	}
	if err := h.updatePasswordUC.Execute(ctx, ucInput); err != nil {
		h.handleUpdateError(w, r, err, ecResult.EmailConfirmation.Email)
		return
	}

	// メール確認IDクッキーを削除
	h.sessionMgr.DeleteEmailConfirmationID(w, r)

	// セッションクッキーを削除（既存セッションを無効化）
	h.sessionMgr.DeleteSessionCookie(w, r)

	// フラッシュメッセージを設定
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "flash_password_reset_success"))

	// ログインページにリダイレクト
	http.Redirect(w, r, "/sign_in", http.StatusFound)
}

// handleUpdateError はパスワード更新処理のエラーを処理する
func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, email string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		h.renderEditForm(w, r, ve)
		return
	}

	slog.ErrorContext(ctx, "パスワードの更新に失敗", "error", err, "email", email)
	ve := model.NewValidationError()
	ve.AddGlobal(templates.T(ctx, "error_password_update_failed"))
	h.renderEditForm(w, r, ve)
}

// renderEditForm はパスワード更新フォームを再表示する
func (h *Handler) renderEditForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError) {
	ctx := r.Context()
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := password_page.EditPageData{
		CSRFToken:  csrfToken,
		FormErrors: ve,
		Flash:      nil,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_edit_title")
	meta.SetOGURL(h.cfg, "/password/edit")

	content := password_page.Edit(data)
	layout := layouts.Simple(meta, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
