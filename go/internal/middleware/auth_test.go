package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// setupAuthTest はテスト用のAuthミドルウェアをセットアップする
func setupAuthTest(t *testing.T) (*Auth, *testutil.SessionBuilder, string) {
	t.Helper()

	_, tx := testutil.SetupTx(t)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("authtestuser").
		Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	sessionBuilder := testutil.NewSessionBuilder(t, tx).
		WithActorID(actorID)
	token := "test-auth-token-12345"
	sessionBuilder.WithToken(token).Build()

	// リポジトリを作成
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))

	// 設定を作成
	cfg := &config.Config{
		CookieDomain:    "localhost",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// セッションマネージャーを作成
	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, profileRepo, cfg)

	// Authミドルウェアを作成
	auth := NewAuth(sessionMgr)

	return auth, sessionBuilder, token
}

func TestRequireAuth_認証済みの場合(t *testing.T) {
	t.Parallel()

	auth, _, token := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// コンテキストからユーザーを取得できることを確認
		user := UserFromContext(r.Context())
		if user == nil {
			t.Error("コンテキストにユーザーが設定されていない")
		}

		// コンテキストからアクターを取得できることを確認
		actor := ActorFromContext(r.Context())
		if actor == nil {
			t.Error("コンテキストにアクターが設定されていない")
		}

		// コンテキストからプロフィールを取得できることを確認
		profile := ProfileFromContext(r.Context())
		if profile == nil {
			t.Error("コンテキストにプロフィールが設定されていない")
		} else if profile.Atname != "authtestuser" {
			t.Errorf("ProfileFromContext().Atname = %q, want %q", profile.Atname, "authtestuser")
		}

		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: token,
	})
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if !handlerCalled {
		t.Error("次のハンドラーが呼び出されなかった")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRequireAuth_未認証の場合(t *testing.T) {
	t.Parallel()

	auth, _, _ := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成 (クッキーなし)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if handlerCalled {
		t.Error("次のハンドラーが呼び出されるべきではない")
	}
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("リダイレクト先が不正: got %s, want /sign_in", location)
	}
}

func TestRequireAuth_無効なトークンの場合(t *testing.T) {
	t.Parallel()

	auth, _, _ := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成 (無効なトークン)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "invalid-token",
	})
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if handlerCalled {
		t.Error("次のハンドラーが呼び出されるべきではない")
	}
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("リダイレクト先が不正: got %s, want /sign_in", location)
	}
}

func TestRequireNoAuth_認証済みの場合(t *testing.T) {
	t.Parallel()

	auth, _, token := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: token,
	})
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireNoAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if handlerCalled {
		t.Error("次のハンドラーが呼び出されるべきではない")
	}
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %s, want /", location)
	}
}

func TestRequireNoAuth_未認証の場合(t *testing.T) {
	t.Parallel()

	auth, _, _ := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成 (クッキーなし)
	req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireNoAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if !handlerCalled {
		t.Error("次のハンドラーが呼び出されなかった")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSetUser_認証済みの場合(t *testing.T) {
	t.Parallel()

	auth, _, token := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	var userFromCtx *model.User
	var actorFromCtx *model.Actor
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		userFromCtx = UserFromContext(r.Context())
		actorFromCtx = ActorFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: token,
	})
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.SetUser(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if !handlerCalled {
		t.Error("次のハンドラーが呼び出されなかった")
	}
	if userFromCtx == nil {
		t.Error("コンテキストにユーザーが設定されていない")
	}
	if actorFromCtx == nil {
		t.Error("コンテキストにアクターが設定されていない")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSetUser_未認証の場合(t *testing.T) {
	t.Parallel()

	auth, _, _ := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	var userFromCtx *model.User
	var actorFromCtx *model.Actor
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		userFromCtx = UserFromContext(r.Context())
		actorFromCtx = ActorFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成 (クッキーなし)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.SetUser(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if !handlerCalled {
		t.Error("次のハンドラーが呼び出されなかった")
	}
	if userFromCtx != nil {
		t.Error("未認証なのにコンテキストにユーザーが設定されている")
	}
	if actorFromCtx != nil {
		t.Error("未認証なのにコンテキストにアクターが設定されている")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestUserFromContext_ユーザーが設定されていない場合(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := UserFromContext(ctx)

	if user != nil {
		t.Error("ユーザーが設定されていないのにnilではない")
	}
}

func TestActorFromContext_アクターが設定されていない場合(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	actor := ActorFromContext(ctx)

	if actor != nil {
		t.Error("アクターが設定されていないのにnilではない")
	}
}

func TestProfileFromContext_プロフィールが設定されていない場合(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	profile := ProfileFromContext(ctx)

	if profile != nil {
		t.Error("プロフィールが設定されていないのにnilではない")
	}
}
