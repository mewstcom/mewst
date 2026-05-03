package password_reset

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Create はパスワードリセット処理を実行する (POST /password_reset)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	turnstileToken := r.FormValue("cf-turnstile-response")

	valid, err := h.turnstileVerifier.Verify(ctx, turnstileToken)
	if err != nil {
		slog.WarnContext(ctx, "Turnstile検証でエラー", "error", err)
	}
	if !valid {
		ve := model.NewValidationError()
		ve.AddGlobal(i18n.T(ctx, "error_turnstile_failed"))
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderPasswordResetForm(w, r, ve, email)
		return
	}

	ucResult, err := h.createPasswordResetUC.Execute(ctx, usecase.CreatePasswordResetInput{
		Email:  email,
		Locale: i18n.GetLocale(ctx),
	})
	if err != nil {
		var ve *model.ValidationError
		if errors.As(err, &ve) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			h.renderPasswordResetForm(w, r, ve, email)
			return
		}
		slog.ErrorContext(ctx, "パスワードリセット処理に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.sessionMgr.SetEmailConfirmationID(w, ucResult.EmailConfirmation.ID)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_password_reset_email_sent"))
	http.Redirect(w, r, "/email_confirmation", http.StatusFound)
}
