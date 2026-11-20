package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// GetLinkUsecaseはcanonical URLでリンクを取得する読み取りユースケース。
type GetLinkUsecase struct {
	linkRepo *repository.LinkRepository
}

// NewGetLinkUsecaseはGetLinkUsecaseを生成する。
func NewGetLinkUsecase(linkRepo *repository.LinkRepository) *GetLinkUsecase {
	return &GetLinkUsecase{linkRepo: linkRepo}
}

// GetLinkInputはリンク取得の入力パラメータ。
type GetLinkInput struct {
	CanonicalURL string
}

// GetLinkOutputはリンク取得の結果。
type GetLinkOutput struct {
	// Linkはcanonical URLに一致するリンクが無い場合nil。
	Link *model.Link
}

// Executeは指定canonical URLのリンクを返す。リンクの不在はここでは
// エラーとせずLink = nilとして返す。呼び出し側はこの取得を表示目的で使い、
// URLが未知の場合はフォールバックする (例: 投稿フォームの422再描画では
// hidden inputのみを残す) ため。
func (uc *GetLinkUsecase) Execute(ctx context.Context, input GetLinkInput) (*GetLinkOutput, error) {
	link, err := uc.linkRepo.FindByCanonicalURL(ctx, input.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("リンクの取得に失敗: %w", err)
	}

	return &GetLinkOutput{Link: link}, nil
}
