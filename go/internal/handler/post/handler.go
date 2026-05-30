// Package post provides HTTP handlers for posts.
// [Ja] Package post は投稿関連の HTTP ハンドラーを提供します。
package post

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	postpages "github.com/mewstcom/mewst/go/internal/templates/pages/post"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Handler is the HTTP handler for post-related endpoints.
// [Ja] Handler は投稿関連の HTTP ハンドラー。
type Handler struct {
	cfg          *config.Config
	flashMgr     *session.FlashManager
	createPostUC *usecase.CreatePostUsecase
}

// NewHandler creates a new Handler.
// [Ja] NewHandler は新しい Handler を作成する。
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	createPostUC *usecase.CreatePostUsecase,
) *Handler {
	return &Handler{
		cfg:          cfg,
		flashMgr:     flashMgr,
		createPostUC: createPostUC,
	}
}

// renderNewForm renders the new post form. It is shared by New (first render)
// and Create (re-render on a validation failure). The current profile and CSRF
// token come from the context populated by RequireAuth and the CSRF middleware.
// content and canonicalURL echo the submitted values back so a failed submit
// keeps the body and the attached link card (both empty on first render). Only
// Create sets a 422 status before calling this; New leaves the default 200.
//
// [Ja] renderNewForm は新規投稿フォームを描画する。New (初回表示) と Create
// (バリデーション失敗時の再表示) の両方から共通利用する。現在プロフィールと CSRF
// トークンは RequireAuth と CSRF ミドルウェアが context に格納する。content と
// canonicalURL は送信値をエコーバックし、送信失敗時に本文と紐付けたリンクカードを
// 保持する (初回表示時はいずれも空)。422 を設定するのは Create のバリデーション失敗
// 時のみで、New はデフォルトの 200 を使う。
func (h *Handler) renderNewForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, content, canonicalURL string) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)
	profile := middleware.ProfileFromContext(ctx)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "post_new_title")

	data := layouts.DefaultLayoutData{
		Meta:   meta,
		Navbar: viewmodel.NewNavbar(profile, viewmodel.NavbarItemNew),
	}
	formContent := postpages.New(postpages.NewPageData{
		CSRFToken:    csrfToken,
		FormErrors:   ve,
		Content:      content,
		CanonicalURL: canonicalURL,
	})

	if err := layouts.Default(data, formContent).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
