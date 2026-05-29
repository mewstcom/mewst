package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestPostRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	profileID := testutil.NewProfileBuilder(t, tx).Build()
	oauthApplicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()

	repo := repository.NewPostRepository(testutil.QueriesWithTx(tx))

	publishedAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	post, err := repo.Create(ctx, repository.CreatePostInput{
		ProfileID:          profileID,
		Content:            "hello world",
		PublishedAt:        publishedAt,
		OauthApplicationID: oauthApplicationID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if post.ProfileID != profileID {
		t.Errorf("post.ProfileID = %v, want %v", post.ProfileID, profileID)
	}
	if post.Content != "hello world" {
		t.Errorf("post.Content = %v, want hello world", post.Content)
	}
	if post.OauthApplicationID != oauthApplicationID {
		t.Errorf("post.OauthApplicationID = %v, want %v", post.OauthApplicationID, oauthApplicationID)
	}
	if !post.PublishedAt.Equal(publishedAt) {
		t.Errorf("post.PublishedAt = %v, want %v", post.PublishedAt, publishedAt)
	}
	if post.DiscardedAt != nil {
		t.Errorf("post.DiscardedAt = %v, want nil", post.DiscardedAt)
	}

	// 作成した投稿がDBに保存され、FindByIDで取得できることを確認
	found, err := repo.FindByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found == nil {
		t.Fatal("FindByID() = nil, want post")
	}
	if found.ID != post.ID {
		t.Errorf("found.ID = %v, want %v", found.ID, post.ID)
	}
}

func TestPostRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	profileID := testutil.NewProfileBuilder(t, tx).Build()
	oauthApplicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()
	publishedAt := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	postID := testutil.NewPostBuilder(t, tx).
		WithProfileID(profileID).
		WithOauthApplicationID(oauthApplicationID).
		WithContent("findbyid content").
		WithPublishedAt(publishedAt).
		Build()

	repo := repository.NewPostRepository(testutil.QueriesWithTx(tx))

	t.Run("存在する投稿を取得できる", func(t *testing.T) {
		post, err := repo.FindByID(ctx, postID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if post == nil {
			t.Fatal("FindByID() = nil, want post")
		}
		if post.ID != postID {
			t.Errorf("post.ID = %v, want %v", post.ID, postID)
		}
		if post.ProfileID != profileID {
			t.Errorf("post.ProfileID = %v, want %v", post.ProfileID, profileID)
		}
		if post.Content != "findbyid content" {
			t.Errorf("post.Content = %v, want findbyid content", post.Content)
		}
		if post.OauthApplicationID != oauthApplicationID {
			t.Errorf("post.OauthApplicationID = %v, want %v", post.OauthApplicationID, oauthApplicationID)
		}
		if !post.PublishedAt.Equal(publishedAt) {
			t.Errorf("post.PublishedAt = %v, want %v", post.PublishedAt, publishedAt)
		}
	})

	t.Run("存在しない投稿はnilを返す", func(t *testing.T) {
		nonExistentID := testutil.NewPostBuilder(t, tx).
			WithProfileID(profileID).
			WithOauthApplicationID(oauthApplicationID).
			Build()
		_, err := tx.Exec("DELETE FROM posts WHERE id = $1", uuid.UUID(nonExistentID))
		if err != nil {
			t.Fatalf("投稿削除に失敗: %v", err)
		}

		post, err := repo.FindByID(ctx, nonExistentID)
		if err != nil {
			t.Errorf("FindByID() error = %v, want nil", err)
		}
		if post != nil {
			t.Errorf("FindByID() post = %v, want nil", post)
		}
	})
}
