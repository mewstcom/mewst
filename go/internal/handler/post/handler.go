// Package post provides HTTP handlers for posts.
// [Ja] Package post は投稿関連の HTTP ハンドラーを提供します。
package post

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
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
	getLinkUC    *usecase.GetLinkUsecase
}

// NewHandler creates a new Handler.
// [Ja] NewHandler は新しい Handler を作成する。
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	createPostUC *usecase.CreatePostUsecase,
	getLinkUC *usecase.GetLinkUsecase,
) *Handler {
	return &Handler{
		cfg:          cfg,
		flashMgr:     flashMgr,
		createPostUC: createPostUC,
		getLinkUC:    getLinkUC,
	}
}

// renderNewForm renders the new post form. It is shared by New (first render)
// and Create (re-render on a validation failure). The current profile and CSRF
// token come from the context populated by RequireAuth and the CSRF middleware.
// content and canonicalURL echo the submitted values back so a failed submit
// keeps the body and the attached link card; attachedLink carries the resolved
// link card to re-render inside #link-form (all empty / nil on first render).
// Only Create sets a 422 status before calling this; New leaves the default 200.
//
// [Ja] renderNewForm は新規投稿フォームを描画する。New (初回表示) と Create
// (バリデーション失敗時の再表示) の両方から共通利用する。現在プロフィールと CSRF
// トークンは RequireAuth と CSRF ミドルウェアが context に格納する。content と
// canonicalURL は送信値をエコーバックし、送信失敗時に本文と紐付けたリンクカードを
// 保持する。attachedLink は #link-form 内に再描画する解決済みリンクカードを運ぶ
// (初回表示時はいずれも空 / nil)。422 を設定するのは Create のバリデーション失敗
// 時のみで、New はデフォルトの 200 を使う。
func (h *Handler) renderNewForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, content, canonicalURL string, attachedLink *viewmodel.Link) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "post_new_title")

	// /new is a focused compose screen, so it uses the navbar-less layouts.Simple
	// (form centered, no navbar) rather than the shared authenticated layout. The
	// cancel affordance lives inside the form's action row (see the page template),
	// so this handler only supplies its /home fallback via NewPageData.BackHref.
	//
	// [Ja] /new は集中作成画面のため、共通の認証後レイアウトではなく navbar を持たない
	// layouts.Simple (フォーム中央寄せ・navbar なし) を使う。キャンセル導線はフォームの
	// 操作行の中に置く (ページテンプレート参照) ため、このハンドラーは NewPageData.BackHref
	// でそのフォールバック先 (/home) を渡すだけでよい。
	formContent := postpages.New(postpages.NewPageData{
		CSRFToken:    csrfToken,
		FormErrors:   ve,
		Content:      content,
		CanonicalURL: canonicalURL,
		AttachedLink: attachedLink,
		BackHref:     templates.HomePath(),
	})

	if err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, formContent).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// lookupAttachedLink resolves the echoed canonical URL into a link card view
// model for the 422 re-render, so the attached link stays visible and removable
// instead of surviving only as an invisible hidden input. The lookup is
// best-effort: an unknown URL or a lookup failure falls back to nil (the bare
// hidden input still keeps the attachment), because failing the whole re-render
// over a cosmetic card would hide the validation errors the user needs to see.
//
// [Ja] lookupAttachedLink はエコーバックする canonical URL を 422 再描画用の
// リンクカード view model に解決する。これにより紐付けたリンクが不可視の hidden
// input としてだけ残るのではなく、カードとして見え削除ボタンで外せる状態を保つ。
// 解決はベストエフォートとし、未知の URL や取得失敗時は nil にフォールバックする
// (紐付け自体は hidden input が保持する)。装飾であるカードのために再描画全体を
// 失敗させると、ユーザーが見るべきバリデーションエラーまで隠れてしまうため。
func (h *Handler) lookupAttachedLink(ctx context.Context, canonicalURL string) *viewmodel.Link {
	if canonicalURL == "" {
		return nil
	}

	output, err := h.getLinkUC.Execute(ctx, usecase.GetLinkInput{CanonicalURL: canonicalURL})
	if err != nil {
		slog.WarnContext(ctx, "再描画用リンクの取得に失敗", "error", err, "canonical_url", canonicalURL)
		return nil
	}
	if output.Link == nil {
		return nil
	}

	link := viewmodel.NewLink(output.Link)
	return &link
}
