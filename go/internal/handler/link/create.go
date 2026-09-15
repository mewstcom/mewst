package link

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	linkpages "github.com/mewstcom/mewst/go/internal/templates/pages/link"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Createは送信されたURLをリンクに解決し、紐付け済みリンクカードの
// フラグメントを返す (POST /links)。1リクエストごとにサーバーが外部サイトへの
// 取得 (最大5ホップ・各10秒・5 MiB) を行うため、先にprofile単位の
// レートリミットを適用する。レートリミット超過時とバリデーションエラー
// (不正なURL・取得失敗) 時は422でプロンプトのフラグメントを再描画し、
// それ以外のエラーは500とする。RailsのLinks::CreateControllerに対応し、
// 取得・再利用・作成のフローはFetchLinkMetadataUsecaseが担う。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	targetURL := r.FormValue("target_url")

	// リクエスト主体はRequireAuthが供給する現在閲覧者のプロフィール。
	// プロフィールが無いのは認証ミドルウェアが想定どおり動いていないことを
	// 意味するため、レートリミットをスキップせず内部エラーとして扱う。
	profile := middleware.ProfileFromContext(ctx)
	if profile == nil {
		slog.ErrorContext(ctx, "現在プロフィールがcontextに存在しない")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := h.checkRateLimit(ctx, profile.ID); err != nil {
		if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
			ve := model.NewValidationError()
			ve.AddField("target_url", i18n.T(ctx, "validation_rate_limit_exceeded"))
			w.WriteHeader(http.StatusUnprocessableEntity)
			h.renderNewFragment(w, r, ve, targetURL)
			return
		}
		slog.ErrorContext(ctx, "レート制限チェックでエラー", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	output, err := h.fetchLinkMetadataUC.Execute(ctx, usecase.FetchLinkMetadataInput{
		TargetURL: targetURL,
	})
	if err != nil {
		var ve *model.ValidationError
		if errors.As(err, &ve) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			h.renderNewFragment(w, r, ve, targetURL)
			return
		}
		slog.ErrorContext(ctx, "リンクの作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := linkpages.CreatePageData{
		Link: viewmodel.NewLink(output.Link),
	}
	if err := linkpages.Create(data).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// checkRateLimitはリンク作成のprofile単位レートリミットを適用する。
// 上限 (10回/分) は正当な利用 (投稿のたびにリンクカードを付ける) では届かない
// 値としつつ、単一profileがサーバーに外部サイト取得を行わせる頻度を抑える。
// キーは "create_link:" プレフィックスでスコープし、将来別エンドポイントが
// profile単位のリミットを導入してもカウンターが混ざらないようにする。
func (h *Handler) checkRateLimit(ctx context.Context, profileID model.ProfileID) error {
	return h.rateLimiter.Allow(ctx, ratelimit.CheckInput{
		Key:    fmt.Sprintf("create_link:profile:%s", profileID),
		Limit:  10,
		Window: time.Minute,
	})
}
