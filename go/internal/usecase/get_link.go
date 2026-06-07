package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// GetLinkUsecase is a read usecase that looks up a link by its canonical URL.
// [Ja] GetLinkUsecase は canonical URL でリンクを取得する読み取りユースケース。
type GetLinkUsecase struct {
	linkRepo *repository.LinkRepository
}

// NewGetLinkUsecase creates a GetLinkUsecase.
// [Ja] NewGetLinkUsecase は GetLinkUsecase を生成する。
func NewGetLinkUsecase(linkRepo *repository.LinkRepository) *GetLinkUsecase {
	return &GetLinkUsecase{linkRepo: linkRepo}
}

// GetLinkInput is the input for looking up a link.
// [Ja] GetLinkInput はリンク取得の入力パラメータ。
type GetLinkInput struct {
	CanonicalURL string
}

// GetLinkOutput is the result of looking up a link.
// [Ja] GetLinkOutput はリンク取得の結果。
type GetLinkOutput struct {
	// Link is nil when no link matches the canonical URL.
	// [Ja] Link は canonical URL に一致するリンクが無い場合 nil。
	Link *model.Link
}

// Execute returns the link with the given canonical URL. A missing link is not
// an error here: it is reported as a nil Link because callers use this lookup
// for display purposes and fall back when the URL is unknown (e.g. the post
// form's 422 re-render keeps the bare hidden input).
//
// [Ja] Execute は指定 canonical URL のリンクを返す。リンクの不在はここでは
// エラーとせず Link = nil として返す。呼び出し側はこの取得を表示目的で使い、
// URL が未知の場合はフォールバックする (例: 投稿フォームの 422 再描画では
// hidden input のみを残す) ため。
func (uc *GetLinkUsecase) Execute(ctx context.Context, input GetLinkInput) (*GetLinkOutput, error) {
	link, err := uc.linkRepo.FindByCanonicalURL(ctx, input.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("リンクの取得に失敗: %w", err)
	}

	return &GetLinkOutput{Link: link}, nil
}
