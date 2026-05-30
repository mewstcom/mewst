package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// FanoutPostUsecase fans a published post out to its author's followers by
// enqueuing one AddPostToTimeline job per follower. It mirrors the Rails
// FanoutPostUseCase: followers are enumerated without a kept filter, and the
// discarded-follower check is deferred to the per-follower delivery job
// (AddPostToTimelineUsecase).
//
// [Ja] FanoutPostUsecase は公開された投稿を投稿者のフォロワーへ配信するため、
// フォロワー 1 人につき AddPostToTimeline ジョブを 1 件 enqueue する。Rails の
// FanoutPostUseCase を踏襲し、フォロワーは kept フィルタ無しで列挙し、discard 済み
// フォロワーの除外は配信ジョブ (AddPostToTimelineUsecase) 側に委ねる。
type FanoutPostUsecase struct {
	postRepo   *repository.PostRepository
	followRepo *repository.FollowRepository
	dispatcher *dispatcher.Dispatcher
}

// NewFanoutPostUsecase は FanoutPostUsecase を生成する
func NewFanoutPostUsecase(
	postRepo *repository.PostRepository,
	followRepo *repository.FollowRepository,
	d *dispatcher.Dispatcher,
) *FanoutPostUsecase {
	return &FanoutPostUsecase{
		postRepo:   postRepo,
		followRepo: followRepo,
		dispatcher: d,
	}
}

// FanoutPostInput はタイムライン配信の入力パラメータ
type FanoutPostInput struct {
	PostID model.PostID
}

// Execute enqueues an AddPostToTimeline job for each follower of the post's author.
// [Ja] Execute は投稿者の各フォロワーに対して AddPostToTimeline ジョブを enqueue する。
func (uc *FanoutPostUsecase) Execute(ctx context.Context, input FanoutPostInput) error {
	post, err := uc.postRepo.FindByID(ctx, input.PostID)
	if err != nil {
		return fmt.Errorf("投稿の取得に失敗: %w", err)
	}
	if post == nil {
		// The post no longer exists (e.g. deleted before fanout ran). There is
		// nothing to deliver, so finish without error to avoid pointless retries.
		// [Ja] 投稿が既に存在しない (fanout 実行前に削除された等)。配信対象が無いため、
		// 無駄なリトライを避けてエラーなしで完了する。
		slog.WarnContext(ctx, "fanout 対象の投稿が見つかりません", "post_id", input.PostID.String())
		return nil
	}

	follows, err := uc.followRepo.ListByTargetProfileID(ctx, post.ProfileID)
	if err != nil {
		return fmt.Errorf("フォロワーの取得に失敗: %w", err)
	}

	for _, follow := range follows {
		if err := uc.dispatcher.EnqueueAddPostToTimeline(ctx, follow.SourceProfileID.String(), post.ID.String()); err != nil {
			// Returning the error makes River retry the whole fanout. The
			// downstream AddPostToTimeline insert is idempotent, so re-enqueuing
			// already-delivered followers on retry is safe.
			// [Ja] エラーを返すと River が fanout 全体をリトライする。後段の
			// AddPostToTimeline の insert は冪等なので、配信済みフォロワーを
			// リトライ時に再 enqueue しても安全。
			return fmt.Errorf("タイムライン配信ジョブの enqueue に失敗: %w", err)
		}
	}

	slog.InfoContext(ctx, "投稿のタイムライン配信ジョブを enqueue しました",
		"post_id", post.ID.String(),
		"follower_count", len(follows),
	)
	return nil
}
