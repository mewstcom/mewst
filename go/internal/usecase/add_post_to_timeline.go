package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// AddPostToTimelineUsecaseは1件の投稿を1つのプロフィールのホームタイムラインに
// 冪等に追加する。FanoutPostUsecaseがenqueueするフォロワー単位の配信ジョブであり、
// RailsのAddPostToTimelineJobを踏襲する。
type AddPostToTimelineUsecase struct {
	profileRepo      *repository.ProfileRepository
	postRepo         *repository.PostRepository
	homeTimelineRepo *repository.HomeTimelinePostRepository
}

// NewAddPostToTimelineUsecaseはAddPostToTimelineUsecaseを生成する
func NewAddPostToTimelineUsecase(
	profileRepo *repository.ProfileRepository,
	postRepo *repository.PostRepository,
	homeTimelineRepo *repository.HomeTimelinePostRepository,
) *AddPostToTimelineUsecase {
	return &AddPostToTimelineUsecase{
		profileRepo:      profileRepo,
		postRepo:         postRepo,
		homeTimelineRepo: homeTimelineRepo,
	}
}

// AddPostToTimelineInputはタイムライン追加の入力パラメータ
type AddPostToTimelineInput struct {
	ProfileID model.ProfileID
	PostID    model.PostID
}

// Executeはプロフィールがkept (discardされていない) フォロワーであれば、
// 投稿をそのホームタイムラインに追加する。
func (uc *AddPostToTimelineUsecase) Execute(ctx context.Context, input AddPostToTimelineInput) error {
	profile, err := uc.profileRepo.FindByID(ctx, input.ProfileID)
	if err != nil {
		return fmt.Errorf("プロフィールの取得に失敗: %w", err)
	}
	// discard済み / 存在しないフォロワーはスキップする。Railsの
	// AddPostToTimelineJobのkept.findガードを踏襲する。Railsは例外でジョブを失敗
	// させるが、本実装はエラーなしで完了する: discard済みフォロワーには何も配信せず、
	// リトライしても成功しないため。
	if profile == nil || profile.DiscardedAt != nil {
		slog.InfoContext(ctx, "discard済みまたは存在しないフォロワーのためタイムライン配信をスキップ", "profile_id", input.ProfileID.String())
		return nil
	}

	post, err := uc.postRepo.FindByID(ctx, input.PostID)
	if err != nil {
		return fmt.Errorf("投稿の取得に失敗: %w", err)
	}
	if post == nil {
		slog.WarnContext(ctx, "配信対象の投稿が見つかりません", "post_id", input.PostID.String())
		return nil
	}

	// 投稿のpublished_atをタイムラインのエントリにコピーし、postsへjoinし直さず
	// 並べ替えられるようにする (Railsのadd_post! に揃える)。
	if _, err := uc.homeTimelineRepo.Create(ctx, repository.CreateHomeTimelinePostInput{
		ProfileID:   input.ProfileID,
		PostID:      input.PostID,
		PublishedAt: post.PublishedAt,
	}); err != nil {
		return fmt.Errorf("ホームタイムラインへの追加に失敗: %w", err)
	}

	return nil
}
