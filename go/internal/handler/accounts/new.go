package accounts

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	accounts_page "github.com/mewstcom/mewst/go/internal/templates/pages/accounts"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// New はアカウント作成フォームを表示する (GET /accounts/new)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

	// セッションからemail_confirmation_idを取得
	emailConfirmation, err := h.getVerifiedEmailConfirmation(r)
	if err != nil {
		slog.ErrorContext(ctx, "メール確認の取得に失敗", "error", err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if emailConfirmation == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := accounts_page.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       nil,
		Email:            emailConfirmation.Email,
		Atname:           "",
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "accounts_new_title")
	meta.SetOGURL(h.cfg, r.URL.Path)

	content := accounts_page.New(data)
	layout := layouts.Simple(meta, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// getVerifiedEmailConfirmation はセッションから確認済みのメール確認を取得する
// メール確認が存在しない、未確認、またはイベントがsign_upでない場合はnilを返す
func (h *Handler) getVerifiedEmailConfirmation(r *http.Request) (*model.EmailConfirmation, error) {
	ecID := h.sessionMgr.GetEmailConfirmationID(r)
	if ecID == "" {
		return nil, nil
	}

	ecUUID, err := uuid.Parse(ecID)
	if err != nil {
		return nil, nil
	}

	output, err := h.getSucceededEmailConfirmationUC.Execute(r.Context(), usecase.GetSucceededEmailConfirmationInput{ID: ecUUID})
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if output.EmailConfirmation.Event != model.EmailConfirmationEventSignUp {
		return nil, nil
	}

	return output.EmailConfirmation, nil
}
