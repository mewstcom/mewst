package password

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	passwordpages "github.com/mewstcom/mewst/go/internal/templates/pages/password"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Update はパスワードを更新する (PATCH /password)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// クッキーからemail_confirmation_idを取得
	id, ok := h.sessionMgr.GetEmailConfirmationID(r)
	if !ok {
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

	// アカウント作成 / パスワード更新 / メール変更フローを取り違えてフォームに到達しないための防御。
	// パスワード更新は password_reset イベントのみ受け付ける。
	if ecResult.EmailConfirmation.Event != model.EmailConfirmationEventPasswordReset {
		http.Redirect(w, r, "/", http.StatusFound)
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
	h.sessionMgr.DeleteEmailConfirmationID(w)

	// セッションクッキーを削除（既存セッションを無効化）
	h.sessionMgr.DeleteSessionCookie(w)

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_password_updated"))

	// ログインページにリダイレクト
	http.Redirect(w, r, "/sign_in", http.StatusFound)
}

// handleUpdateError はパスワード更新処理のエラーを処理する
func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, email string) {
	ctx := r.Context()

	var ve *model.ValidationError
	if errors.As(err, &ve) {
		h.renderPasswordEditForm(w, r, ve)
		return
	}

	slog.ErrorContext(ctx, "パスワードの更新に失敗", "error", err, "email", email)
	{
		ve := model.NewValidationError()
		ve.AddGlobal(i18n.T(ctx, "validation_password_update_failed"))
		h.renderPasswordEditForm(w, r, ve)
	}
}

// renderPasswordEditForm はパスワード更新フォームを再表示する
func (h *Handler) renderPasswordEditForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError) {
	ctx := r.Context()
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := passwordpages.EditPageData{
		CSRFToken:  csrfToken,
		FormErrors: ve,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_edit_title")

	content := passwordpages.Edit(data)
	layout := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
