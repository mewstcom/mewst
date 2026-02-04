package email_confirmation

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	email_confirmation_page "github.com/mewstcom/mewst/go/internal/templates/pages/email_confirmation"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// New は確認コード入力フォームを表示する (GET /email_confirmation)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールと設定を設定（テンプレート内での翻訳用）
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

	// 有効な確認レコードを取得
	_, err = h.emailConfirmationRepo.GetActiveByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// 見つからないまたは期限切れの場合はルートにリダイレクト
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
	data := email_confirmation_page.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: session.NewFormErrors(),
		Code:       "",
		Flash:      flash,
	}

	// テンプレートをレンダリング
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "email_confirmation_title")
	meta.SetOGURL(h.cfg, r.URL.Path)

	content := email_confirmation_page.New(data)
	layout := layouts.Simple(meta, content)

	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
