package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestHomeTimelinePostRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	profileID := testutil.NewProfileBuilder(t, tx).Build()
	oauthApplicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()
	publishedAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	postID := testutil.NewPostBuilder(t, tx).
		WithProfileID(profileID).
		WithOauthApplicationID(oauthApplicationID).
		WithPublishedAt(publishedAt).
		Build()

	repo := repository.NewHomeTimelinePostRepository(testutil.QueriesWithTx(tx))

	t.Run("ホームタイムライン投稿を作成できる", func(t *testing.T) {
		htp, err := repo.Create(ctx, repository.CreateHomeTimelinePostInput{
			ProfileID:   profileID,
			PostID:      postID,
			PublishedAt: publishedAt,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if htp.ProfileID != profileID {
			t.Errorf("htp.ProfileID = %v, want %v", htp.ProfileID, profileID)
		}
		if htp.PostID != postID {
			t.Errorf("htp.PostID = %v, want %v", htp.PostID, postID)
		}
		if !htp.PublishedAt.Equal(publishedAt) {
			t.Errorf("htp.PublishedAt = %v, want %v", htp.PublishedAt, publishedAt)
		}
	})

	t.Run("同じprofile_id + post_idで再作成しても冪等に既存行を返す", func(t *testing.T) {
		first, err := repo.Create(ctx, repository.CreateHomeTimelinePostInput{
			ProfileID:   profileID,
			PostID:      postID,
			PublishedAt: publishedAt,
		})
		if err != nil {
			t.Fatalf("Create() (1回目) error = %v", err)
		}

		// On the second call we pass a different published_at, but the existing
		// row (with the original published_at) must come back and no duplicate
		// row must be created.
		// [Ja] 2回目は別の published_at を渡しても、既存行 (元の published_at) が
		// 返り、重複行は作られないことを確認する。
		second, err := repo.Create(ctx, repository.CreateHomeTimelinePostInput{
			ProfileID:   profileID,
			PostID:      postID,
			PublishedAt: publishedAt.Add(24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("Create() (2回目) error = %v", err)
		}

		if second.ID != first.ID {
			t.Errorf("second.ID = %v, want %v (冪等に同一行が返るべき)", second.ID, first.ID)
		}
		if !second.PublishedAt.Equal(publishedAt) {
			t.Errorf("second.PublishedAt = %v, want %v (既存値を保持すべき)", second.PublishedAt, publishedAt)
		}

		var count int
		if err := tx.QueryRow(
			"SELECT COUNT(*) FROM home_timeline_posts WHERE profile_id = $1 AND post_id = $2",
			uuid.UUID(profileID), uuid.UUID(postID),
		).Scan(&count); err != nil {
			t.Fatalf("COUNT クエリに失敗: %v", err)
		}
		if count != 1 {
			t.Errorf("home_timeline_posts の行数 = %d, want 1", count)
		}
	})
}
