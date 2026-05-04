package password_test

import (
	"database/sql"
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
	handler "github.com/mewstcom/mewst/go/internal/handler/password"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// setupTestHandler はテスト用のハンドラーとテストデータをセットアップする
func setupTestHandler(t *testing.T, tx *sql.Tx) (*handler.Handler, *config.Config) {
	t.Helper()

	cfg := testutil.NewTestConfig(t)

	// トランザクションを使用するリポジトリを作成
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	getSucceededEmailConfirmationUC := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmRepo)
	passwordUpdateValidator := validator.NewPasswordUpdateValidator()
	updatePasswordUC := usecase.NewUpdatePasswordUsecase(passwordUpdateValidator, userRepo)

	h := handler.NewHandler(cfg, sessionMgr, flashMgr, getSucceededEmailConfirmationUC, updatePasswordUC)

	return h, cfg
}
