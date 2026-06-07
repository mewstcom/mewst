// Package link provides HTTP handlers for link cards.
// [Ja] Package link はリンクカード関連の HTTP ハンドラーを提供します。
package link

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	linkpages "github.com/mewstcom/mewst/go/internal/templates/pages/link"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Handler is the HTTP handler for link card endpoints. Both endpoints return
// htmx HTML fragments swapped into the #link-form container of the post form,
// not full pages.
//
// [Ja] Handler はリンクカード関連の HTTP ハンドラー。どちらのエンドポイントも
// フルページではなく、投稿フォームの #link-form コンテナにスワップされる htmx の
// HTML フラグメントを返す。
type Handler struct {
	fetchLinkMetadataUC *usecase.FetchLinkMetadataUsecase
	rateLimiter         *ratelimit.Limiter
}

// NewHandler creates a new Handler.
// [Ja] NewHandler は新しい Handler を作成する。
func NewHandler(fetchLinkMetadataUC *usecase.FetchLinkMetadataUsecase, rateLimiter *ratelimit.Limiter) *Handler {
	return &Handler{
		fetchLinkMetadataUC: fetchLinkMetadataUC,
		rateLimiter:         rateLimiter,
	}
}

// renderNewFragment renders the add-link-card prompt fragment. It is shared by
// New (first render) and Create (re-render on a validation failure). The CSRF
// token comes from the context populated by the CSRF middleware; targetURL is
// echoed back as a hidden field so the button resubmits the same URL. Only
// Create sets a 422 status before calling this; New leaves the default 200.
//
// [Ja] renderNewFragment はリンクカード追加プロンプトのフラグメントを描画する。
// New (初回表示) と Create (バリデーション失敗時の再表示) の両方から共通利用する。
// CSRF トークンは CSRF ミドルウェアが context に格納する。targetURL は hidden
// フィールドとしてエコーバックし、ボタンが同じ URL を再送信できるようにする。
// 422 を設定するのは Create のバリデーション失敗時のみで、New はデフォルトの
// 200 を使う。
func (h *Handler) renderNewFragment(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, targetURL string) {
	ctx := r.Context()

	data := linkpages.NewPageData{
		CSRFToken:   middleware.GetCSRFTokenFromContext(ctx),
		TargetURL:   targetURL,
		HostAndPath: viewmodel.ShortenHostAndPath(targetURL),
		FormErrors:  ve,
	}

	if err := linkpages.New(data).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
