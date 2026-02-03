package password

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Update はパスワードを更新する (PATCH /password)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	// 確認済みのメール確認レコードを取得
	emailConfirmation, err := h.emailConfirmationRepo.GetSucceededByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// 未確認または期限切れの場合はルートにリダイレクト
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		slog.ErrorContext(ctx, "メール確認レコードの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フォームデータを取得
	input := UpdateValidatorInput{
		Password: r.FormValue("password"),
	}

	// フォームバリデーション
	validator := NewUpdateValidator()
	result := validator.Validate(ctx, input)
	if result.FormErrors.HasErrors() {
		h.renderEditForm(w, ctx, csrfToken, result.FormErrors)
		return
	}

	// パスワードを更新
	ucInput := usecase.UpdatePasswordInput{
		Email:    emailConfirmation.Email,
		Password: input.Password,
	}
	if err := h.updatePasswordUC.Execute(ctx, ucInput); err != nil {
		slog.ErrorContext(ctx, "パスワードの更新に失敗", "error", err, "email", emailConfirmation.Email)
		formErrors := session.NewFormErrors()
		formErrors.AddGlobalError(templates.T(ctx, "error_password_update_failed"))
		h.renderEditForm(w, ctx, csrfToken, formErrors)
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
