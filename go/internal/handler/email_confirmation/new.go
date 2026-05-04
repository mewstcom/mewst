package email_confirmation

import (
	"errors"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	emailconfirmationpages "github.com/mewstcom/mewst/go/internal/templates/pages/email_confirmation"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// New は確認コード入力フォームを表示する (GET /email_confirmation)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// クッキーからemail_confirmation_idを取得
	id, ok := h.sessionMgr.GetEmailConfirmationID(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// 有効な確認レコードを取得
	_, err := h.getActiveEmailConfirmationUC.Execute(ctx, usecase.GetActiveEmailConfirmationInput{ID: id})
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページデータを作成
	data := emailconfirmationpages.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: nil,
		Code:       "",
		BackURL:    r.URL.Query().Get("back"),
	}

	// テンプレートをレンダリング
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "email_confirmation_new_title")

	content := emailconfirmationpages.New(data)
	layout := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
