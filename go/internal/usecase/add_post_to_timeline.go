package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// AddPostToTimelineUsecase idempotently adds a single post to a single
// profile's home timeline. It is the per-follower delivery job enqueued by
// FanoutPostUsecase and mirrors the Rails AddPostToTimelineJob.
//
// [Ja] AddPostToTimelineUsecase は 1 件の投稿を 1 つのプロフィールのホームタイムラインに
// 冪等に追加する。FanoutPostUsecase が enqueue するフォロワー単位の配信ジョブであり、
// Rails の AddPostToTimelineJob を踏襲する。
type AddPostToTimelineUsecase struct {
	profileRepo      *repository.ProfileRepository
	postRepo         *repository.PostRepository
	homeTimelineRepo *repository.HomeTimelinePostRepository
}

// NewAddPostToTimelineUsecase は AddPostToTimelineUsecase を生成する
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

// AddPostToTimelineInput はタイムライン追加の入力パラメータ
type AddPostToTimelineInput struct {
	ProfileID model.ProfileID
	PostID    model.PostID
}

// Execute adds the post to the profile's home timeline if the profile is a kept
// (non-discarded) follower.
// [Ja] Execute はプロフィールが kept (discard されていない) フォロワーであれば、
// 投稿をそのホームタイムラインに追加する。
func (uc *AddPostToTimelineUsecase) Execute(ctx context.Context, input AddPostToTimelineInput) error {
	profile, err := uc.profileRepo.FindByID(ctx, input.ProfileID)
	if err != nil {
		return fmt.Errorf("プロフィールの取得に失敗: %w", err)
	}
	// Skip discarded or missing followers, mirroring the Rails
	// AddPostToTimelineJob guard (ProfileRecord.kept.find). Unlike Rails, which
	// raises and fails the job, we finish without error: a discarded follower
	// simply receives nothing, and retrying would never succeed.
	// [Ja] discard 済み / 存在しないフォロワーはスキップする。Rails の
	// AddPostToTimelineJob の kept.find ガードを踏襲する。Rails は例外でジョブを失敗
	// させるが、本実装はエラーなしで完了する: discard 済みフォロワーには何も配信せず、
	// リトライしても成功しないため。
	if profile == nil || profile.DiscardedAt != nil {
		slog.InfoContext(ctx, "discard 済みまたは存在しないフォロワーのためタイムライン配信をスキップ", "profile_id", input.ProfileID.String())
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

	// Copy the post's published_at into the timeline entry so the timeline can
	// be ordered without joining back to posts (matches Rails add_post!).
	// [Ja] 投稿の published_at をタイムラインのエントリにコピーし、posts へ join し直さず
	// 並べ替えられるようにする (Rails の add_post! に揃える)。
	if _, err := uc.homeTimelineRepo.Create(ctx, repository.CreateHomeTimelinePostInput{
		ProfileID:   input.ProfileID,
		PostID:      input.PostID,
		PublishedAt: post.PublishedAt,
	}); err != nil {
		return fmt.Errorf("ホームタイムラインへの追加に失敗: %w", err)
	}

	return nil
}
