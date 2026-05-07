package account

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	accountpages "github.com/mewstcom/mewst/go/internal/templates/pages/account"
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
		var ae *model.AppError
		if errors.As(err, &ae) {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound:
				http.Redirect(w, r, "/", http.StatusFound)
				return
			default:
				slog.ErrorContext(ctx, ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
		slog.ErrorContext(ctx, "メール確認の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// アカウント作成 / パスワード更新 / メール変更フローを取り違えてフォームに到達しないための防御。
	// アカウント作成は sign_up イベントのみ受け付ける。
	if ucResult.EmailConfirmation.Event != model.EmailConfirmationEventSignUp {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := accountpages.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       nil,
		Email:            ucResult.EmailConfirmation.Email,
		Atname:           "",
		BackURL:          r.URL.Query().Get("back"),
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "account_new_title")

	content := accountpages.New(data)
	layout := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
