// Package linkはリンクカード関連のHTTPハンドラーを提供します。
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

// Handlerはリンクカード関連のHTTPハンドラー。どちらのエンドポイントも
// フルページではなく、投稿フォームの #link-formコンテナにスワップされるhtmxの
// HTMLフラグメントを返す。
type Handler struct {
	fetchLinkMetadataUC *usecase.FetchLinkMetadataUsecase
	rateLimiter         *ratelimit.Limiter
}

// NewHandlerは新しいHandlerを作成する。
func NewHandler(fetchLinkMetadataUC *usecase.FetchLinkMetadataUsecase, rateLimiter *ratelimit.Limiter) *Handler {
	return &Handler{
		fetchLinkMetadataUC: fetchLinkMetadataUC,
		rateLimiter:         rateLimiter,
	}
}

// renderNewFragmentはリンクカード追加プロンプトのフラグメントを描画する。
// New (初回表示) とCreate (バリデーション失敗時の再表示) の両方から共通利用する。
// CSRFトークンはCSRFミドルウェアがcontextに格納する。targetURLはhidden
// フィールドとしてエコーバックし、ボタンが同じURLを再送信できるようにする。
// 422を設定するのはCreateのバリデーション失敗時のみで、Newはデフォルトの
// 200を使う。
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
