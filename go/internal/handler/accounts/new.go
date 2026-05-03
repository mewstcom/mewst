package accounts

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	accountspages "github.com/mewstcom/mewst/go/internal/templates/pages/accounts"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// New はアカウント作成フォームを表示する (GET /accounts/new)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// クッキーからemail_confirmation_idを取得
	id, ok := h.sessionMgr.GetEmailConfirmationID(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// 確認済みのメール確認レコードを取得
	ucResult, err := h.getSucceededEmailConfirmationUC.Execute(ctx, usecase.GetSucceededEmailConfirmationInput{ID: id})
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		slog.ErrorContext(ctx, "メール確認の取得に失敗", "error", err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// アカウント作成 / パスワード更新 / メール変更フローを取り違えてフォームに到達しないための防御。
	// アカウント作成は sign_up イベントのみ受け付ける。
	if ucResult.EmailConfirmation.Event != model.EmailConfirmationEventSignUp {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := accountspages.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       nil,
		Email:            ucResult.EmailConfirmation.Email,
		Atname:           "",
		BackURL:          r.URL.Query().Get("back"),
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "accounts_new_title")

	content := accountspages.New(data)
	layout := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
