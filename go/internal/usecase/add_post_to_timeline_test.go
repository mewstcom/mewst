package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestAddPostToTimelineUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("ホームタイムラインに投稿を追加する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		profileID := testutil.NewProfileBuilder(t, tx).Build()
		authorID := testutil.NewProfileBuilder(t, tx).Build()
		oauthAppID := testutil.NewOauthApplicationBuilder(t, tx).Build()
		publishedAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
		postID := testutil.NewPostBuilder(t, tx).
			WithProfileID(authorID).
			WithOauthApplicationID(oauthAppID).
			WithPublishedAt(publishedAt).
			Build()

		uc := usecase.NewAddPostToTimelineUsecase(
			repository.NewProfileRepository(testutil.QueriesWithTx(tx)),
			repository.NewPostRepository(testutil.QueriesWithTx(tx)),
			repository.NewHomeTimelinePostRepository(testutil.QueriesWithTx(tx)),
		)

		if err := uc.Execute(ctx, usecase.AddPostToTimelineInput{ProfileID: profileID, PostID: postID}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		var gotPublishedAt time.Time
		err := tx.QueryRow(
			`SELECT published_at FROM home_timeline_posts WHERE profile_id = $1 AND post_id = $2`,
			uuid.UUID(profileID), uuid.UUID(postID),
		).Scan(&gotPublishedAt)
		if err != nil {
			t.Fatalf("home_timeline_posts の取得に失敗: %v", err)
		}
		// The timeline entry must copy the post's published_at.
		// [Ja] タイムラインのエントリは投稿の published_at を複製していること。
		if !gotPublishedAt.Equal(publishedAt) {
			t.Errorf("published_at = %v, want %v", gotPublishedAt, publishedAt)
		}
	})

	t.Run("冪等: 同じ投稿で再実行しても重複しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		profileID := testutil.NewProfileBuilder(t, tx).Build()
		authorID := testutil.NewProfileBuilder(t, tx).Build()
		oauthAppID := testutil.NewOauthApplicationBuilder(t, tx).Build()
		postID := testutil.NewPostBuilder(t, tx).
			WithProfileID(authorID).
			WithOauthApplicationID(oauthAppID).
			Build()

		uc := usecase.NewAddPostToTimelineUsecase(
			repository.NewProfileRepository(testutil.QueriesWithTx(tx)),
			repository.NewPostRepository(testutil.QueriesWithTx(tx)),
			repository.NewHomeTimelinePostRepository(testutil.QueriesWithTx(tx)),
		)

		input := usecase.AddPostToTimelineInput{ProfileID: profileID, PostID: postID}
		if err := uc.Execute(ctx, input); err != nil {
			t.Fatalf("Execute() (1回目) error = %v", err)
		}
		if err := uc.Execute(ctx, input); err != nil {
			t.Fatalf("Execute() (2回目) error = %v", err)
		}

		var count int
		err := tx.QueryRow(
			`SELECT count(*) FROM home_timeline_posts WHERE profile_id = $1 AND post_id = $2`,
			uuid.UUID(profileID), uuid.UUID(postID),
		).Scan(&count)
		if err != nil {
			t.Fatalf("home_timeline_posts の件数取得に失敗: %v", err)
		}
		if count != 1 {
			t.Errorf("home_timeline_posts の件数 = %d, want 1", count)
		}
	})

	t.Run("discard 済みフォロワーには追加しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		discardedProfileID := testutil.NewProfileBuilder(t, tx).
			WithDiscardedAt(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).
			Build()
		authorID := testutil.NewProfileBuilder(t, tx).Build()
		oauthAppID := testutil.NewOauthApplicationBuilder(t, tx).Build()
		postID := testutil.NewPostBuilder(t, tx).
			WithProfileID(authorID).
			WithOauthApplicationID(oauthAppID).
			Build()

		uc := usecase.NewAddPostToTimelineUsecase(
			repository.NewProfileRepository(testutil.QueriesWithTx(tx)),
			repository.NewPostRepository(testutil.QueriesWithTx(tx)),
			repository.NewHomeTimelinePostRepository(testutil.QueriesWithTx(tx)),
		)

		// A discarded follower must be skipped without error (Rails kept.find guard).
		// [Ja] discard 済みフォロワーはエラーなくスキップされること (Rails の kept.find ガード)。
		if err := uc.Execute(ctx, usecase.AddPostToTimelineInput{ProfileID: discardedProfileID, PostID: postID}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		var count int
		err := tx.QueryRow(
			`SELECT count(*) FROM home_timeline_posts WHERE profile_id = $1 AND post_id = $2`,
			uuid.UUID(discardedProfileID), uuid.UUID(postID),
		).Scan(&count)
		if err != nil {
			t.Fatalf("home_timeline_posts の件数取得に失敗: %v", err)
		}
		if count != 0 {
			t.Errorf("home_timeline_posts の件数 = %d, want 0", count)
		}
	})

	t.Run("フォロワーが存在しなければエラーなくスキップする", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		authorID := testutil.NewProfileBuilder(t, tx).Build()
		oauthAppID := testutil.NewOauthApplicationBuilder(t, tx).Build()
		postID := testutil.NewPostBuilder(t, tx).
			WithProfileID(authorID).
			WithOauthApplicationID(oauthAppID).
			Build()

		uc := usecase.NewAddPostToTimelineUsecase(
			repository.NewProfileRepository(testutil.QueriesWithTx(tx)),
			repository.NewPostRepository(testutil.QueriesWithTx(tx)),
			repository.NewHomeTimelinePostRepository(testutil.QueriesWithTx(tx)),
		)

		if err := uc.Execute(ctx, usecase.AddPostToTimelineInput{ProfileID: model.ProfileID(uuid.New()), PostID: postID}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	})
}
