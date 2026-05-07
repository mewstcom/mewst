package sign_in

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/clientip"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/redirect"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Create はログイン処理を実行する (POST /sign_in)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	turnstileToken := r.FormValue("cf-turnstile-response")
	backURL := r.FormValue("back")

	valid, err := h.turnstileVerifier.Verify(ctx, turnstileToken)
	if err != nil {
		slog.WarnContext(ctx, "Turnstile検証でエラー", "error", err)
	}
	if !valid {
		ve := model.NewValidationError()
		ve.AddGlobal(i18n.T(ctx, "validation_turnstile_failed"))
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderSignInForm(w, r, ve, email, backURL)
		return
	}

	output, err := h.signInUC.Execute(ctx, usecase.CreateSignInInput{
		Email:     email,
		Password:  password,
		IPAddress: clientip.GetClientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		var ve *model.ValidationError
		if errors.As(err, &ve) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			h.renderSignInForm(w, r, ve, email, backURL)
			return
		}
		var ae *model.AppError
		if errors.As(err, &ae) {
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		slog.ErrorContext(ctx, "ログイン処理に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.sessionMgr.SetSessionCookie(w, output.Token)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_sign_in_success"))

	http.Redirect(w, r, redirect.GetSafeRedirectURL(backURL), http.StatusFound)
}
