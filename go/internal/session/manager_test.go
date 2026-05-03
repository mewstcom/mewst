package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestManager_GetSessionToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    "localhost",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	manager := NewManager(nil, nil, nil, cfg)

	tests := []struct {
		name     string
		cookie   *http.Cookie
		expected string
	}{
		{
			name:     "クッキーがない場合は空文字を返す",
			cookie:   nil,
			expected: "",
		},
		{
			name: "クッキーがある場合はトークンを返す",
			cookie: &http.Cookie{
				Name:  CookieName,
				Value: "test-token-123",
			},
			expected: "test-token-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			got := manager.GetSessionToken(req)
			if got != tt.expected {
				t.Errorf("GetSessionToken() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestManager_GetCurrentUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("session-test@example.com").
		Build()

	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("sessiontestuser").
		Build()

	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	testToken := "test-session-token-abc123"
	testutil.NewSessionBuilder(t, tx).
		WithActorID(actorID).
		WithToken(testToken).
		Build()

	cfg := &config.Config{
		CookieDomain:    "localhost",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))

	manager := NewManager(sessionRepo, actorRepo, userRepo, cfg)

	tests := []struct {
		name     string
		token    string
		wantUser bool
		wantErr  bool
	}{
		{
			name:     "有効なトークンでユーザーを取得できる",
			token:    testToken,
			wantUser: true,
			wantErr:  false,
		},
		{
			name:     "無効なトークンではnilを返す",
			token:    "invalid-token",
			wantUser: false,
			wantErr:  false,
		},
		{
			name:     "トークンがない場合はnilを返す",
			token:    "",
			wantUser: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.token != "" {
				req.AddCookie(&http.Cookie{
					Name:  CookieName,
					Value: tt.token,
				})
			}

			user, err := manager.GetCurrentUser(context.Background(), req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantUser && user == nil {
				t.Error("GetCurrentUser() = nil, want user")
			}
			if !tt.wantUser && user != nil {
				t.Errorf("GetCurrentUser() = %v, want nil", user)
			}

			if tt.wantUser && user != nil {
				if user.Email != "session-test@example.com" {
					t.Errorf("GetCurrentUser().Email = %q, want %q", user.Email, "session-test@example.com")
				}
			}
		})
	}
}

func TestManager_GetCurrentActor(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("actor-test@example.com").
		Build()

	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("actortestuser").
		Build()

	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	testToken := "test-actor-token-xyz789"
	testutil.NewSessionBuilder(t, tx).
		WithActorID(actorID).
		WithToken(testToken).
		Build()

	cfg := &config.Config{
		CookieDomain:    "localhost",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))

	manager := NewManager(sessionRepo, actorRepo, userRepo, cfg)

	tests := []struct {
		name      string
		token     string
		wantActor bool
		wantErr   bool
	}{
		{
			name:      "有効なトークンでアクターを取得できる",
			token:     testToken,
			wantActor: true,
			wantErr:   false,
		},
		{
			name:      "無効なトークンではnilを返す",
			token:     "invalid-actor-token",
			wantActor: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.token != "" {
				req.AddCookie(&http.Cookie{
					Name:  CookieName,
					Value: tt.token,
				})
			}

			actor, err := manager.GetCurrentActor(context.Background(), req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentActor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantActor && actor == nil {
				t.Error("GetCurrentActor() = nil, want actor")
			}
			if !tt.wantActor && actor != nil {
				t.Errorf("GetCurrentActor() = %v, want nil", actor)
			}

			if tt.wantActor && actor != nil {
				if actor.ID != actorID {
					t.Errorf("GetCurrentActor().ID = %v, want %v", actor.ID, actorID)
				}
			}
		})
	}
}

func TestManager_SetSessionCookie(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    "example.com",
		SessionSecure:   true,
		SessionHTTPOnly: true,
	}

	manager := NewManager(nil, nil, nil, cfg)

	rr := httptest.NewRecorder()
	manager.SetSessionCookie(rr, "test-token")

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("SetSessionCookie() set %d cookies, want 1", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != CookieName {
		t.Errorf("Cookie.Name = %q, want %q", cookie.Name, CookieName)
	}
	if cookie.Value != "test-token" {
		t.Errorf("Cookie.Value = %q, want %q", cookie.Value, "test-token")
	}
	if !cookie.Secure {
		t.Error("Cookie.Secure = false, want true")
	}
	if !cookie.HttpOnly {
		t.Error("Cookie.HttpOnly = false, want true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("Cookie.SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
	}
	if cookie.MaxAge != MaxAge {
		t.Errorf("Cookie.MaxAge = %d, want %d", cookie.MaxAge, MaxAge)
	}
}

func TestManager_DeleteSessionCookie(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    "example.com",
		SessionSecure:   true,
		SessionHTTPOnly: true,
	}

	manager := NewManager(nil, nil, nil, cfg)

	rr := httptest.NewRecorder()
	manager.DeleteSessionCookie(rr)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("DeleteSessionCookie() set %d cookies, want 1", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != CookieName {
		t.Errorf("Cookie.Name = %q, want %q", cookie.Name, CookieName)
	}
	if cookie.Value != "" {
		t.Errorf("Cookie.Value = %q, want empty string", cookie.Value)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("Cookie.MaxAge = %d, want -1", cookie.MaxAge)
	}
}

func TestManager_IsLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("loggedin-test@example.com").
		Build()

	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("loggedintestuser").
		Build()

	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	testToken := "test-loggedin-token"
	testutil.NewSessionBuilder(t, tx).
		WithActorID(actorID).
		WithToken(testToken).
		Build()

	cfg := &config.Config{
		CookieDomain:    "localhost",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))

	manager := NewManager(sessionRepo, actorRepo, userRepo, cfg)

	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{
			name:  "有効なトークンでログイン済みと判定される",
			token: testToken,
			want:  true,
		},
		{
			name:  "無効なトークンでは未ログインと判定される",
			token: "invalid-token",
			want:  false,
		},
		{
			name:  "トークンがない場合は未ログインと判定される",
			token: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.token != "" {
				req.AddCookie(&http.Cookie{
					Name:  CookieName,
					Value: tt.token,
				})
			}

			got := manager.IsLoggedIn(context.Background(), req)
			if got != tt.want {
				t.Errorf("IsLoggedIn() = %v, want %v", got, tt.want)
			}
		})
	}
}
