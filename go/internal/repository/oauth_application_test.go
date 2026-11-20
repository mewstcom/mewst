package repository_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestOauthApplicationRepository_FindByUID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	oauthApplicationID := testutil.NewOauthApplicationBuilder(t, tx).
		WithName("Mewst for Web").
		WithUID(model.MewstWebUID).
		Build()

	repo := repository.NewOauthApplicationRepository(testutil.QueriesWithTx(tx))

	t.Run("uidでOAuthアプリケーションを取得できる", func(t *testing.T) {
		app, err := repo.FindByUID(ctx, model.MewstWebUID)
		if err != nil {
			t.Fatalf("FindByUID()のエラー = %v", err)
		}
		if app == nil {
			t.Fatal("FindByUID() = nil、OAuthアプリケーションを期待")
		}
		if app.ID != oauthApplicationID {
			t.Errorf("app.ID = %v、期待値 = %v", app.ID, oauthApplicationID)
		}
		if app.UID != model.MewstWebUID {
			t.Errorf("app.UID = %v、期待値 = %v", app.UID, model.MewstWebUID)
		}
		if app.Name != "Mewst for Web" {
			t.Errorf("app.Name = %v、期待値 = Mewst for Web", app.Name)
		}
	})

	t.Run("存在しないuidはnilを返す", func(t *testing.T) {
		app, err := repo.FindByUID(ctx, "nonexistent-uid")
		if err != nil {
			t.Errorf("FindByUID()のエラー = %v、期待値 = nil", err)
		}
		if app != nil {
			t.Errorf("FindByUID()のapp = %v、期待値 = nil", app)
		}
	})
}
