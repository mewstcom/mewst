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

// CreatePostUsecaseは投稿の作成とその副作用を統括するオーケストレーションUseCaseで、
// RailsのCreatePostUseCaseを踏襲する。本文をバリデーションし、投稿をmewst-webのOAuth
// アプリケーションに紐付けたうえで、1トランザクション内で投稿をinsertし、必要ならリンク
// カードを紐付け、投稿者のlast_post_atを更新し、投稿者自身のホームタイムラインに追加する。
// トランザクションのコミット後、フォロワーのタイムラインへ配信するFanoutPostジョブを
// enqueueする。
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

// NewCreatePostUsecaseはCreatePostUsecaseを生成する
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

// CreatePostInputは投稿作成の入力パラメータ
type CreatePostInput struct {
	// AuthorProfileIDは投稿者のプロフィールID
	AuthorProfileID model.ProfileID
	// Contentは投稿本文
	Content string
	// CanonicalURLはリンクカードとして紐付けるlinkのcanonical_url (空なら紐付けない)
	CanonicalURL string
}

// CreatePostOutputは投稿作成の出力パラメータ
type CreatePostOutput struct {
	Post *model.Post
}

// Executeは投稿を作成し、副作用 (リンク紐付け・last_post_at更新・自タイムライン追加・fanout) を発生させる。
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
		return nil, fmt.Errorf("mewst-web OAuthアプリケーションの取得に失敗: %w", err)
	}
	if oauthApp == nil {
		// mewst-webは常に存在すべきシードデータであり、欠落はユーザー入力エラーではなく
		// 設定エラー。
		return nil, &model.AppError{
			Code:     model.AppErrCodeInternal,
			UserMsg:  "Internal server error",
			Internal: fmt.Errorf("mewst-web OAuthアプリケーション (uid=%s) が存在しません", model.MewstWebUID),
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

	// コミット後にフォロワーへの配信ジョブをenqueueする。enqueueをトランザクション
	// 外に出すのは、配信ジョブがコミット済みの投稿を前提とするため (トランザクション内で
	// enqueueしてロールバックすると、存在しない投稿の配信ジョブが残ってしまう)。
	if err := uc.dispatcher.EnqueueFanoutPost(ctx, post.ID.String()); err != nil {
		// 投稿は既にコミット済みのため、enqueueの失敗でリクエスト全体を失敗させない。
		// ログを残して継続する (投稿自体は成功しており、フォロワーのタイムラインにこの投稿が
		// 反映されないだけ)。
		slog.ErrorContext(ctx, "fanoutジョブのenqueueに失敗", "post_id", post.ID.String(), "error", err)
	}

	return &CreatePostOutput{Post: post}, nil
}

// createPostは投稿・リンク紐付け・last_post_at更新・自タイムライン追加を1トランザクションで行う
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

	// 投稿のpublished_atを使い、last_post_atを投稿と完全に一致させる
	// (Railsのupdate_last_post_time!(time: post.published_at) に揃える)。
	if err := profileRepo.UpdateLastPostAt(ctx, input.AuthorProfileID, post.PublishedAt); err != nil {
		return nil, fmt.Errorf("last_post_atの更新に失敗: %w", err)
	}

	// 投稿を投稿者自身のホームタイムラインに追加し、fanoutを待たずに即時表示されるように
	// する (Railsのhome_timeline.add_post! に揃える)。
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
