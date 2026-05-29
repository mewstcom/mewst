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
			t.Fatalf("FindByUID() error = %v", err)
		}
		if app == nil {
			t.Fatal("FindByUID() = nil, want oauth application")
		}
		if app.ID != oauthApplicationID {
			t.Errorf("app.ID = %v, want %v", app.ID, oauthApplicationID)
		}
		if app.UID != model.MewstWebUID {
			t.Errorf("app.UID = %v, want %v", app.UID, model.MewstWebUID)
		}
		if app.Name != "Mewst for Web" {
			t.Errorf("app.Name = %v, want Mewst for Web", app.Name)
		}
	})

	t.Run("存在しないuidはnilを返す", func(t *testing.T) {
		app, err := repo.FindByUID(ctx, "nonexistent-uid")
		if err != nil {
			t.Errorf("FindByUID() error = %v, want nil", err)
		}
		if app != nil {
			t.Errorf("FindByUID() app = %v, want nil", app)
		}
	})
}
