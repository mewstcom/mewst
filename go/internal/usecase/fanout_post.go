package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// FanoutPostUsecaseは公開された投稿を投稿者のフォロワーへ配信するため、
// フォロワー1人につきAddPostToTimelineジョブを1件enqueueする。Railsの
// FanoutPostUseCaseを踏襲し、フォロワーはkeptフィルタ無しで列挙し、discard済み
// フォロワーの除外は配信ジョブ (AddPostToTimelineUsecase) 側に委ねる。
type FanoutPostUsecase struct {
	postRepo   *repository.PostRepository
	followRepo *repository.FollowRepository
	dispatcher *dispatcher.Dispatcher
}

// NewFanoutPostUsecaseはFanoutPostUsecaseを生成する
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

// FanoutPostInputはタイムライン配信の入力パラメータ
type FanoutPostInput struct {
	PostID model.PostID
}

// Executeは投稿者の各フォロワーに対してAddPostToTimelineジョブをenqueueする。
func (uc *FanoutPostUsecase) Execute(ctx context.Context, input FanoutPostInput) error {
	post, err := uc.postRepo.FindByID(ctx, input.PostID)
	if err != nil {
		return fmt.Errorf("投稿の取得に失敗: %w", err)
	}
	if post == nil {
		// 投稿が既に存在しない (fanout実行前に削除された等)。配信対象が無いため、
		// 無駄なリトライを避けてエラーなしで完了する。
		slog.WarnContext(ctx, "fanout対象の投稿が見つかりません", "post_id", input.PostID.String())
		return nil
	}

	follows, err := uc.followRepo.ListByTargetProfileID(ctx, post.ProfileID)
	if err != nil {
		return fmt.Errorf("フォロワーの取得に失敗: %w", err)
	}

	for _, follow := range follows {
		if err := uc.dispatcher.EnqueueAddPostToTimeline(ctx, follow.SourceProfileID.String(), post.ID.String()); err != nil {
			// エラーを返すとRiverがfanout全体をリトライする。後段の
			// AddPostToTimelineのinsertは冪等なので、配信済みフォロワーを
			// リトライ時に再enqueueしても安全。
			return fmt.Errorf("タイムライン配信ジョブのenqueueに失敗: %w", err)
		}
	}

	slog.InfoContext(ctx, "投稿のタイムライン配信ジョブをenqueueしました",
		"post_id", post.ID.String(),
		"follower_count", len(follows),
	)
	return nil
}
