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

// Create resolves the submitted URL into a link and returns the attached-link-
// card fragment (POST /links). Each request makes the server fetch external
// sites (up to 5 hops, 10 seconds and 5 MiB each), so a per-profile rate limit
// is applied first. On a rate-limit hit or a validation error (invalid URL or
// fetch failure) it re-renders the prompt fragment with status 422; any other
// error becomes a 500. Mirrors the Rails Links::CreateController, whose fetch +
// reuse + create flow lives in FetchLinkMetadataUsecase.
//
// [Ja] Create は送信された URL をリンクに解決し、紐付け済みリンクカードの
// フラグメントを返す (POST /links)。1 リクエストごとにサーバーが外部サイトへの
// 取得 (最大 5 ホップ・各 10 秒・5 MiB) を行うため、先に profile 単位の
// レートリミットを適用する。レートリミット超過時とバリデーションエラー
// (不正な URL・取得失敗) 時は 422 でプロンプトのフラグメントを再描画し、
// それ以外のエラーは 500 とする。Rails の Links::CreateController に対応し、
// 取得・再利用・作成のフローは FetchLinkMetadataUsecase が担う。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	targetURL := r.FormValue("target_url")

	// The requester is the current viewer's profile, supplied by RequireAuth.
	// A missing profile means the auth middleware did not run as expected, so
	// treat it as an internal error rather than skipping the rate limit.
	//
	// [Ja] リクエスト主体は RequireAuth が供給する現在閲覧者のプロフィール。
	// プロフィールが無いのは認証ミドルウェアが想定どおり動いていないことを
	// 意味するため、レートリミットをスキップせず内部エラーとして扱う。
	profile := middleware.ProfileFromContext(ctx)
	if profile == nil {
		slog.ErrorContext(ctx, "現在プロフィールが context に存在しない")
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

// checkRateLimit enforces the per-profile rate limit for link creation. The
// limit (10 requests/min) is high enough that legitimately attaching a link
// card to every post never hits it, while capping how often a single profile
// can make the server fetch external sites. The key is scoped with a
// "create_link:" prefix so a future profile-keyed limit on another endpoint
// gets its own counter.
//
// [Ja] checkRateLimit はリンク作成の profile 単位レートリミットを適用する。
// 上限 (10 回/分) は正当な利用 (投稿のたびにリンクカードを付ける) では届かない
// 値としつつ、単一 profile がサーバーに外部サイト取得を行わせる頻度を抑える。
// キーは "create_link:" プレフィックスでスコープし、将来別エンドポイントが
// profile 単位のリミットを導入してもカウンターが混ざらないようにする。
func (h *Handler) checkRateLimit(ctx context.Context, profileID model.ProfileID) error {
	return h.rateLimiter.Allow(ctx, ratelimit.CheckInput{
		Key:    fmt.Sprintf("create_link:profile:%s", profileID),
		Limit:  10,
		Window: time.Minute,
	})
}
