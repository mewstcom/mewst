package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// CreatePostUsecase orchestrates creating a post and its side effects, mirroring
// the Rails CreatePostUseCase: it validates the body, attributes the post to the
// mewst-web OAuth application, then in a single transaction inserts the post,
// optionally links a card, bumps the author's last_post_at, and adds the post to
// the author's own home timeline. After the transaction commits it enqueues a
// FanoutPost job to deliver the post to followers' timelines.
//
// [Ja] CreatePostUsecase は投稿の作成とその副作用を統括するオーケストレーション UseCase で、
// Rails の CreatePostUseCase を踏襲する。本文をバリデーションし、投稿を mewst-web の OAuth
// アプリケーションに紐付けたうえで、1 トランザクション内で投稿を insert し、必要ならリンク
// カードを紐付け、投稿者の last_post_at を更新し、投稿者自身のホームタイムラインに追加する。
// トランザクションのコミット後、フォロワーのタイムラインへ配信する FanoutPost ジョブを
// enqueue する。
type CreatePostUsecase struct {
	db               *sql.DB
	postValidator    *validator.PostCreateValidator
	oauthAppRepo     *repository.OauthApplicationRepository
	linkRepo         *repository.LinkRepository
	postRepo         *repository.PostRepository
	postLinkRepo     *repository.PostLinkRepository
	profileRepo      *repository.ProfileRepository
	homeTimelineRepo *repository.HomeTimelinePostRepository
	dispatcher       *dispatcher.Dispatcher
}

// NewCreatePostUsecase は CreatePostUsecase を生成する
func NewCreatePostUsecase(
	db *sql.DB,
	postValidator *validator.PostCreateValidator,
	oauthAppRepo *repository.OauthApplicationRepository,
	linkRepo *repository.LinkRepository,
	postRepo *repository.PostRepository,
	postLinkRepo *repository.PostLinkRepository,
	profileRepo *repository.ProfileRepository,
	homeTimelineRepo *repository.HomeTimelinePostRepository,
	d *dispatcher.Dispatcher,
) *CreatePostUsecase {
	return &CreatePostUsecase{
		db:               db,
		postValidator:    postValidator,
		oauthAppRepo:     oauthAppRepo,
		linkRepo:         linkRepo,
		postRepo:         postRepo,
		postLinkRepo:     postLinkRepo,
		profileRepo:      profileRepo,
		homeTimelineRepo: homeTimelineRepo,
		dispatcher:       d,
	}
}

// CreatePostInput は投稿作成の入力パラメータ
type CreatePostInput struct {
	// AuthorProfileID は投稿者のプロフィール ID
	AuthorProfileID model.ProfileID
	// Content は投稿本文
	Content string
	// CanonicalURL はリンクカードとして紐付ける link の canonical_url (空なら紐付けない)
	CanonicalURL string
}

// CreatePostOutput は投稿作成の出力パラメータ
type CreatePostOutput struct {
	Post *model.Post
}

// Execute creates a post and triggers its side effects (link attachment,
// last_post_at update, the author's own timeline insert, and fanout).
//
// [Ja] Execute は投稿を作成し、副作用 (リンク紐付け・last_post_at 更新・自タイムライン追加・fanout) を発生させる。
func (uc *CreatePostUsecase) Execute(ctx context.Context, input CreatePostInput) (*CreatePostOutput, error) {
	// 1. バリデーション (トランザクション外)
	if err := uc.postValidator.Validate(ctx, validator.PostCreateValidatorInput{
		Content: input.Content,
	}); err != nil {
		return nil, err
	}

	// 2. 投稿に紐付けるデータの取得 (トランザクション外)
	oauthApp, err := uc.oauthAppRepo.FindByUID(ctx, model.MewstWebUID)
	if err != nil {
		return nil, fmt.Errorf("mewst-web OAuth アプリケーションの取得に失敗: %w", err)
	}
	if oauthApp == nil {
		// mewst-web is seed data that must always exist; its absence is a
		// configuration error rather than user input error.
		// [Ja] mewst-web は常に存在すべきシードデータであり、欠落はユーザー入力エラーではなく
		// 設定エラー。
		return nil, &model.AppError{
			Code:     model.AppErrCodeInternal,
			UserMsg:  "Internal server error",
			Internal: fmt.Errorf("mewst-web OAuth アプリケーション (uid=%s) が存在しません", model.MewstWebUID),
		}
	}

	var link *model.Link
	if input.CanonicalURL != "" {
		link, err = uc.linkRepo.FindByCanonicalURL(ctx, input.CanonicalURL)
		if err != nil {
			return nil, fmt.Errorf("リンクの取得に失敗: %w", err)
		}
	}

	currentTime := time.Now()

	// 3. ビジネスロジック + 永続化
	post, err := uc.createPost(ctx, input, oauthApp.ID, link, currentTime)
	if err != nil {
		return nil, err
	}

	// 4. Enqueue the follower-delivery job after the commit. It is kept outside
	// the transaction because the delivery job assumes an already-committed post
	// (enqueuing inside the tx and then rolling back would leave a delivery job
	// for a post that does not exist).
	//
	// [Ja] コミット後にフォロワーへの配信ジョブを enqueue する。enqueue をトランザクション
	// 外に出すのは、配信ジョブがコミット済みの投稿を前提とするため (トランザクション内で
	// enqueue してロールバックすると、存在しない投稿の配信ジョブが残ってしまう)。
	if err := uc.dispatcher.EnqueueFanoutPost(ctx, post.ID.String()); err != nil {
		// The post is already committed, so a failed enqueue must not fail the
		// whole request. Log it and continue; followers simply miss this post
		// on their timelines (the post itself is created successfully).
		// [Ja] 投稿は既にコミット済みのため、enqueue の失敗でリクエスト全体を失敗させない。
		// ログを残して継続する (投稿自体は成功しており、フォロワーのタイムラインにこの投稿が
		// 反映されないだけ)。
		slog.ErrorContext(ctx, "fanout ジョブの enqueue に失敗", "post_id", post.ID.String(), "error", err)
	}

	return &CreatePostOutput{Post: post}, nil
}

// createPost は投稿・リンク紐付け・last_post_at 更新・自タイムライン追加を 1 トランザクションで行う
func (uc *CreatePostUsecase) createPost(
	ctx context.Context,
	input CreatePostInput,
	oauthApplicationID model.OauthApplicationID,
	link *model.Link,
	currentTime time.Time,
) (*model.Post, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	postRepo := uc.postRepo.WithTx(tx)
	postLinkRepo := uc.postLinkRepo.WithTx(tx)
	profileRepo := uc.profileRepo.WithTx(tx)
	homeTimelineRepo := uc.homeTimelineRepo.WithTx(tx)

	post, err := postRepo.Create(ctx, repository.CreatePostInput{
		ProfileID:          input.AuthorProfileID,
		Content:            input.Content,
		PublishedAt:        currentTime,
		OauthApplicationID: oauthApplicationID,
	})
	if err != nil {
		return nil, fmt.Errorf("投稿の作成に失敗: %w", err)
	}

	if link != nil {
		if _, err := postLinkRepo.Create(ctx, repository.CreatePostLinkInput{
			PostID: post.ID,
			LinkID: link.ID,
		}); err != nil {
			return nil, fmt.Errorf("リンクカードの紐付けに失敗: %w", err)
		}
	}

	// Use the post's published_at so last_post_at matches the post exactly
	// (mirrors Rails update_last_post_time!(time: post.published_at)).
	// [Ja] 投稿の published_at を使い、last_post_at を投稿と完全に一致させる
	// (Rails の update_last_post_time!(time: post.published_at) に揃える)。
	if err := profileRepo.UpdateLastPostAt(ctx, input.AuthorProfileID, post.PublishedAt); err != nil {
		return nil, fmt.Errorf("last_post_at の更新に失敗: %w", err)
	}

	// Add the post to the author's own home timeline so it appears immediately
	// without waiting for fanout (mirrors Rails home_timeline.add_post!).
	// [Ja] 投稿を投稿者自身のホームタイムラインに追加し、fanout を待たずに即時表示されるように
	// する (Rails の home_timeline.add_post! に揃える)。
	if _, err := homeTimelineRepo.Create(ctx, repository.CreateHomeTimelinePostInput{
		ProfileID:   input.AuthorProfileID,
		PostID:      post.ID,
		PublishedAt: post.PublishedAt,
	}); err != nil {
		return nil, fmt.Errorf("自身のホームタイムラインへの追加に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return post, nil
}
