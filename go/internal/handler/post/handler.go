// Package postは投稿関連のHTTPハンドラーを提供します。
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

// Handlerは投稿関連のHTTPハンドラー。
type Handler struct {
	cfg          *config.Config
	flashMgr     *session.FlashManager
	createPostUC *usecase.CreatePostUsecase
	getLinkUC    *usecase.GetLinkUsecase
}

// draftStorageKeyPrefixはlocalStorageの下書きキーを機能ごとに名前空間化し、
// ユーザー別の本文下書き (web/post_draft.ts) が他の保存キーと衝突しないようにする。
// 投稿者のプロフィールIDを後ろに付けてユーザー別にスコープする。
const draftStorageKeyPrefix = "post_draft:"

// NewHandlerは新しいHandlerを作成する。
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

// renderNewFormは新規投稿フォームを描画する。New (初回表示) とCreate
// (バリデーション失敗時の再表示) の両方から共通利用する。現在プロフィールとCSRF
// トークンはRequireAuthとCSRFミドルウェアがcontextに格納する。contentと
// canonicalURLは送信値をエコーバックし、送信失敗時に本文と紐付けたリンクカードを
// 保持する。attachedLinkは #link-form内に再描画する解決済みリンクカードを運ぶ
// (初回表示時はいずれも空 / nil)。422を設定するのはCreateのバリデーション失敗
// 時のみで、Newはデフォルトの200を使う。
func (h *Handler) renderNewForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, content, canonicalURL string, attachedLink *viewmodel.Link) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "post_new_title")

	// /newはlayouts.Centeredを使い、集中した中央寄せの作成カラムを保ちながら、
	// 共通の認証後レイアウトと同じグローバルnavbar (PC右上・モバイル下部中央) を表示する。
	// これはRailsの投稿フォーム周辺で利用できるナビゲーションに揃う。/newはnewページの
	// ためnavbarはnewをアクティブ表示する。ページ単位の戻る導線はnavbarを補完する
	// (navbar = 全体ナビ、戻る = 直前ページへ戻る)。このハンドラーはNewPageData.BackHref
	// で戻るリンクのフォールバック先 (/home) を渡す。
	profile := middleware.ProfileFromContext(ctx)

	navbar := viewmodel.NewNavbar(profile, viewmodel.NavbarItemNew)

	// localStorageの下書きキーを現在の投稿者にスコープし、共有端末で前ユーザーの
	// 未送信下書きが次のユーザーに復元されないようにする。プロフィールが無いのは
	// RequireAuth外で /newを描画したとき (例: 一部のテスト) だけであり、キーを空の
	// ままにすることで共有キーへの書き込みではなく自動保存の無効化とする。
	var draftStorageKey string
	if profile != nil {
		draftStorageKey = draftStorageKeyPrefix + profile.ID.String()
	}

	formContent := postpages.New(postpages.NewPageData{
		CSRFToken:       csrfToken,
		FormErrors:      ve,
		Content:         content,
		CanonicalURL:    canonicalURL,
		AttachedLink:    attachedLink,
		BackHref:        templates.HomePath(),
		DraftStorageKey: draftStorageKey,
	})

	if err := layouts.Centered(layouts.CenteredLayoutData{Meta: meta, Navbar: navbar}, formContent).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// lookupAttachedLinkはエコーバックするcanonical URLを422再描画用の
// リンクカードview modelに解決する。これにより紐付けたリンクが不可視のhidden
// inputとしてだけ残るのではなく、カードとして見え削除ボタンで外せる状態を保つ。
// 解決はベストエフォートとし、未知のURLや取得失敗時はnilにフォールバックする
// (紐付け自体はhidden inputが保持する)。装飾であるカードのために再描画全体を
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
