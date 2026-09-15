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
		t.Fatalf("Create()のエラー = %v", err)
	}

	if post.ProfileID != profileID {
		t.Errorf("post.ProfileID = %v、期待値 = %v", post.ProfileID, profileID)
	}
	if post.Content != "hello world" {
		t.Errorf("post.Content = %v、期待値 = hello world", post.Content)
	}
	if post.OauthApplicationID != oauthApplicationID {
		t.Errorf("post.OauthApplicationID = %v、期待値 = %v", post.OauthApplicationID, oauthApplicationID)
	}
	if !post.PublishedAt.Equal(publishedAt) {
		t.Errorf("post.PublishedAt = %v、期待値 = %v", post.PublishedAt, publishedAt)
	}
	if post.DiscardedAt != nil {
		t.Errorf("post.DiscardedAt = %v、期待値 = nil", post.DiscardedAt)
	}

	// 作成した投稿がDBに保存され、FindByIDで取得できることを確認
	found, err := repo.FindByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if found == nil {
		t.Fatal("FindByID() = nil、ポストを期待")
	}
	if found.ID != post.ID {
		t.Errorf("found.ID = %v、期待値 = %v", found.ID, post.ID)
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
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if post == nil {
			t.Fatal("FindByID() = nil、ポストを期待")
		}
		if post.ID != postID {
			t.Errorf("post.ID = %v、期待値 = %v", post.ID, postID)
		}
		if post.ProfileID != profileID {
			t.Errorf("post.ProfileID = %v、期待値 = %v", post.ProfileID, profileID)
		}
		if post.Content != "findbyid content" {
			t.Errorf("post.Content = %v、期待値 = findbyid content", post.Content)
		}
		if post.OauthApplicationID != oauthApplicationID {
			t.Errorf("post.OauthApplicationID = %v、期待値 = %v", post.OauthApplicationID, oauthApplicationID)
		}
		if !post.PublishedAt.Equal(publishedAt) {
			t.Errorf("post.PublishedAt = %v、期待値 = %v", post.PublishedAt, publishedAt)
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
			t.Errorf("FindByID()のエラー = %v、期待値 = nil", err)
		}
		if post != nil {
			t.Errorf("FindByID()のpost = %v、期待値 = nil", post)
		}
	})
}
