package link

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/model"
	linkpages "github.com/mewstcom/mewst/go/internal/templates/pages/link"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Create resolves the submitted URL into a link and returns the attached-link-
// card fragment (POST /links). On a validation error (invalid URL or fetch
// failure) it re-renders the prompt fragment with status 422; any other error
// becomes a 500. Mirrors the Rails Links::CreateController, whose fetch +
// reuse + create flow lives in FetchLinkMetadataUsecase.
//
// [Ja] Create は送信された URL をリンクに解決し、紐付け済みリンクカードの
// フラグメントを返す (POST /links)。バリデーションエラー (不正な URL・取得失敗)
// 時は 422 でプロンプトのフラグメントを再描画し、それ以外のエラーは 500 とする。
// Rails の Links::CreateController に対応し、取得・再利用・作成のフローは
// FetchLinkMetadataUsecase が担う。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	targetURL := r.FormValue("target_url")

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
