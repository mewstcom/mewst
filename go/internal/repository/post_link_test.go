package repository_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestPostLinkRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	profileID := testutil.NewProfileBuilder(t, tx).Build()
	oauthApplicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()
	postID := testutil.NewPostBuilder(t, tx).
		WithProfileID(profileID).
		WithOauthApplicationID(oauthApplicationID).
		Build()
	linkID := testutil.NewLinkBuilder(t, tx).Build()

	repo := repository.NewPostLinkRepository(testutil.QueriesWithTx(tx))

	t.Run("投稿とリンクの関連付けを作成できる", func(t *testing.T) {
		postLink, err := repo.Create(ctx, repository.CreatePostLinkInput{
			PostID: postID,
			LinkID: linkID,
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}

		if postLink.PostID != postID {
			t.Errorf("postLink.PostID = %v、期待値 = %v", postLink.PostID, postID)
		}
		if postLink.LinkID != linkID {
			t.Errorf("postLink.LinkID = %v、期待値 = %v", postLink.LinkID, linkID)
		}
	})
}
