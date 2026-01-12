package password

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/internal/middleware"
	"github.com/mewstcom/mewst/internal/repository"
	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
	"github.com/mewstcom/mewst/internal/templates/layouts"
	password_page "github.com/mewstcom/mewst/internal/templates/pages/password"
	"github.com/mewstcom/mewst/internal/viewmodel"
)

// Edit は新しいパスワード入力フォームを表示する (GET /password/edit)
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールと設定を設定
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, h.cfg)

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
	_, err = h.emailConfirmationRepo.GetSucceededByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// 未確認または期限切れの場合はルートにリダイレクト
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// フラッシュメッセージを取得
	flash := h.sessionMgr.GetFlashFromCookie(w, r)

	// ページデータを作成
	data := password_page.EditPageData{
		CSRFToken:  csrfToken,
		FormErrors: session.NewFormErrors(),
		Flash:      flash,
	}

	// テンプレートをレンダリング
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_edit_title")
	meta.SetOGURL(h.cfg, r.URL.Path)

	content := password_page.Edit(data)
	layout := layouts.Simple(meta, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// renderEditForm は新しいパスワード入力フォームを再表示する
func (h *Handler) renderEditForm(w http.ResponseWriter, ctx context.Context, csrfToken string, formErrors *session.FormErrors) {
	data := password_page.EditPageData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
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
