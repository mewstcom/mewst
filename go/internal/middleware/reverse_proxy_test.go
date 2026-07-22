package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// stubFeatureFlagChecker is a featureFlagChecker that returns canned values for tests.
// [Ja] stubFeatureFlagChecker はテスト用に固定値を返す featureFlagChecker。
type stubFeatureFlagChecker struct {
	enabled bool
	err     error
}

func (s stubFeatureFlagChecker) IsEnabledForDevice(_ context.Context, _ string, _ string, _ model.FeatureFlagName) (bool, error) {
	return s.enabled, s.err
}

func TestReverseProxyMiddleware_GoHandledPaths(t *testing.T) {
	t.Parallel()

	// モックRailsサーバーを作成
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	// テスト用の設定
	cfg := &config.Config{
		Domain: "mewst-test.com",
	}

	// リバースプロキシミドルウェアを作成
	proxyMiddleware, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	// Go版で処理するハンドラー (ダミー)
	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})

	// ミドルウェアを適用
	handler := proxyMiddleware.Middleware(goHandler)

	// テストケース：Go版で処理するパス
	testCases := []struct {
		name         string
		path         string
		expectedBody string
	}{
		{"静的ファイル", "/static/css/style.css", "Go response"},
		{"ヘルスチェック", "/health", "Go response"},
		{"ログインページ", "/sign_in", "Go response"},
		{"ログイン処理", "/sign_in", "Go response"},
		{"サインアップページ", "/sign_up", "Go response"},
		{"ログアウト", "/sign_out", "Go response"},
		{"アカウント作成フォーム", "/accounts/new", "Go response"},
		{"アカウント作成処理", "/accounts", "Go response"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコードが期待と異なる: got %v want %v", rr.Code, http.StatusOK)
			}

			if rr.Body.String() != tc.expectedBody {
				t.Errorf("レスポンスボディが期待と異なる: got %q want %q", rr.Body.String(), tc.expectedBody)
			}
		})
	}
}

func TestReverseProxyMiddleware_RailsProxiedPaths(t *testing.T) {
	t.Parallel()

	// モックRailsサーバーを作成
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// X-Forwarded-*ヘッダーが設定されていることを確認
		if r.Header.Get("X-Forwarded-Proto") != "https" {
			t.Errorf("X-Forwarded-Protoが設定されていない: got %q", r.Header.Get("X-Forwarded-Proto"))
		}
		if r.Header.Get("X-Forwarded-Host") != "mewst-test.com" {
			t.Errorf("X-Forwarded-Hostが設定されていない: got %q", r.Header.Get("X-Forwarded-Host"))
		}
		// X-Forwarded-ForとX-Real-IPが設定されていることを確認
		if r.Header.Get("X-Forwarded-For") == "" {
			t.Errorf("X-Forwarded-Forが設定されていない")
		}
		if r.Header.Get("X-Real-IP") == "" {
			t.Errorf("X-Real-IPが設定されていない")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	// テスト用の設定
	cfg := &config.Config{
		Domain: "mewst-test.com",
	}

	// リバースプロキシミドルウェアを作成
	proxyMiddleware, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	// Go版で処理するハンドラー (ダミー)
	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})

	// ミドルウェアを適用
	handler := proxyMiddleware.Middleware(goHandler)

	// テストケース：Rails版にプロキシするパス
	testCases := []struct {
		name         string
		path         string
		expectedBody string
	}{
		{"トップページ", "/", "Rails response"},
		{"ユーザープロフィール", "/@username", "Rails response"},
		// "/settings" itself is now handled by Go (see the Settings tests below);
		// its sub-pages stay with Rails, so use one here as the settings example.
		//
		// [Ja] "/settings" 本体は Go 版で処理されるようになった (後述の Settings テストを参照)。
		// サブページは Rails に残るため、設定系の代表としてサブページを使う。
		{"設定サブページ", "/settings/profile", "Rails response"},
		{"投稿ページ", "/posts/123", "Rails response"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコードが期待と異なる: got %v want %v", rr.Code, http.StatusOK)
			}

			if rr.Body.String() != tc.expectedBody {
				t.Errorf("レスポンスボディが期待と異なる: got %q want %q", rr.Body.String(), tc.expectedBody)
			}
		})
	}
}

func TestIsGoHandledPath(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Domain: "mewst-test.com"}
	proxyMiddleware, _ := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)

	testCases := []struct {
		path     string
		expected bool
	}{
		{"/static/css/style.css", true},
		{"/static/js/app.js", true},
		{"/health", true},
		{"/sign_in", true},
		{"/sign_in/", true},
		{"/sign_up", true},
		{"/sign_up/", true},
		{"/sign_out", true},
		{"/sign_out/", true},
		{"/accounts", true},
		{"/accounts/new", true},
		{"/", false},
		{"/@username", false},
		{"/settings", false},
		{"/posts/123", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			actual := proxyMiddleware.isGoHandledPath(tc.path)
			if actual != tc.expected {
				t.Errorf("isGoHandledPath(%q) = %v, want %v", tc.path, actual, tc.expected)
			}
		})
	}
}

func TestReverseProxyMiddleware_ErrorHandling(t *testing.T) {
	t.Parallel()

	// モックRailsサーバーを作成 (常にエラーを返す)
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 接続を即座に閉じる (エラーをシミュレート)
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("Hijackerをサポートしていない")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("Hijackに失敗: %v", err)
		}
		_ = conn.Close()
	}))
	defer railsServer.Close()

	// テスト用の設定
	cfg := &config.Config{
		Domain: "mewst-test.com",
	}

	// リバースプロキシミドルウェアを作成
	proxyMiddleware, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	// Go版で処理するハンドラー (ダミー)
	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})

	// ミドルウェアを適用
	handler := proxyMiddleware.Middleware(goHandler)

	// リクエストを作成
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// エラーハンドリングにより502 Bad Gatewayが返ることを確認
	if rr.Code != http.StatusBadGateway {
		t.Errorf("ステータスコードが期待と異なる: got %v want %v", rr.Code, http.StatusBadGateway)
	}
}

func TestReverseProxyMiddleware_HeaderForwarding(t *testing.T) {
	t.Parallel()

	// モックRailsサーバーを作成 (ヘッダーチェック)
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 各種ヘッダーが転送されていることを確認
		headers := map[string]string{
			"CF-Connecting-IP": "1.2.3.4",
			"Cookie":           "_mewst_session=test_session_id",
		}

		for name, expected := range headers {
			actual := r.Header.Get(name)
			if actual != expected {
				t.Errorf("ヘッダー %s が期待と異なる: got %q want %q", name, actual, expected)
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	// テスト用の設定
	cfg := &config.Config{
		Domain: "mewst-test.com",
	}

	// リバースプロキシミドルウェアを作成
	proxyMiddleware, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	// Go版で処理するハンドラー (ダミー)
	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})

	// ミドルウェアを適用
	handler := proxyMiddleware.Middleware(goHandler)

	// リクエストを作成 (ヘッダーを設定)
	req := httptest.NewRequest("GET", "/posts", nil)
	req.Header.Set("CF-Connecting-IP", "1.2.3.4")
	req.Header.Set("Cookie", "_mewst_session=test_session_id")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが期待と異なる: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestReverseProxyMiddleware_HTTPMethods(t *testing.T) {
	t.Parallel()

	// モックRailsサーバーを作成 (HTTPメソッドを確認)
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HTTPメソッドをレスポンスボディに含める
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Method: " + r.Method))
	}))
	defer railsServer.Close()

	// テスト用の設定
	cfg := &config.Config{
		Domain: "mewst-test.com",
	}

	// リバースプロキシミドルウェアを作成
	proxyMiddleware, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	// Go版で処理するハンドラー (ダミー)
	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})

	// ミドルウェアを適用
	handler := proxyMiddleware.Middleware(goHandler)

	// Various HTTP methods are proxied to Rails. "/settings/profile" is a pure
	// Rails sub-path present in neither goHandledPaths nor goHandledPatterns, so it
	// is proxied to Rails for every method. (Its parent GET /settings is now
	// handled by Go, and POST /posts is handled by Go too, so neither can be used
	// here.)
	//
	// [Ja] 様々な HTTP メソッドが Rails 版にプロキシされることを確認する。
	// "/settings/profile" は goHandledPaths にも goHandledPatterns にも含まれない
	// 純粋な Rails サブパスで、どの HTTP メソッドでも Rails にプロキシされる。
	// (親の GET /settings は Go の処理対象になり、POST /posts も Go の処理対象のため、
	// どちらもここでは使えない。)
	testCases := []struct {
		method       string
		expectedBody string
	}{
		{"GET", "Method: GET"},
		{"POST", "Method: POST"},
		{"PUT", "Method: PUT"},
		{"PATCH", "Method: PATCH"},
		{"DELETE", "Method: DELETE"},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/settings/profile", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコードが期待と異なる: got %v want %v", rr.Code, http.StatusOK)
			}

			if rr.Body.String() != tc.expectedBody {
				t.Errorf("レスポンスボディが期待と異なる: got %q want %q", rr.Body.String(), tc.expectedBody)
			}
		})
	}
}

func TestReverseProxyMiddleware_ensureDeviceToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Domain:        "mewst-test.com",
		CookieDomain:  "mewst-test.com",
		SessionSecure: true,
	}

	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	t.Run("device_token Cookieがない場合は自動生成される", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		m.ensureDeviceToken(rr, req)

		var deviceCookie *http.Cookie
		for _, c := range rr.Result().Cookies() {
			if c.Name == DeviceTokenCookieName {
				deviceCookie = c
				break
			}
		}

		if deviceCookie == nil {
			t.Fatal("device_token Cookieが設定されていない")
		}
		if deviceCookie.Value == "" {
			t.Error("device_token Cookieの値が空")
		}
		if !deviceCookie.HttpOnly {
			t.Error("HttpOnlyが設定されていない")
		}
		if !deviceCookie.Secure {
			t.Error("Secureが設定されていない")
		}
		if deviceCookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("SameSite = %v, want %v", deviceCookie.SameSite, http.SameSiteLaxMode)
		}
		if deviceCookie.Domain != "mewst-test.com" {
			t.Errorf("Domain = %q, want %q", deviceCookie.Domain, "mewst-test.com")
		}
	})

	t.Run("device_token Cookieが既に存在する場合は再生成しない", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: DeviceTokenCookieName, Value: "existing-token"})
		rr := httptest.NewRecorder()

		m.ensureDeviceToken(rr, req)

		for _, c := range rr.Result().Cookies() {
			if c.Name == DeviceTokenCookieName {
				t.Error("既存のdevice_token Cookieがあるのに新しいCookieが設定された")
			}
		}
	})
}

// TestReverseProxyMiddleware_Middleware_DeviceTokenIssuance verifies that the
// device_token cookie is issued only after the Go-handled-path check, so static
// assets and health checks never receive a Set-Cookie.
//
// [Ja] device_token Cookie が Go 処理パスの判定より後で発行され、静的アセットや
// ヘルスチェックには Set-Cookie が付かないことを検証する。
func TestReverseProxyMiddleware_Middleware_DeviceTokenIssuance(t *testing.T) {
	t.Parallel()

	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	cfg := &config.Config{
		Domain:        "mewst-test.com",
		CookieDomain:  "mewst-test.com",
		SessionSecure: true,
	}

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})
	handler := m.Middleware(goHandler)

	hasDeviceTokenCookie := func(rr *httptest.ResponseRecorder) bool {
		for _, c := range rr.Result().Cookies() {
			if c.Name == DeviceTokenCookieName {
				return true
			}
		}
		return false
	}

	t.Run("Go処理パスではdevice_tokenを発行しない", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if hasDeviceTokenCookie(rr) {
			t.Error("Go処理パスでdevice_token Cookieが発行された")
		}
	})

	t.Run("Rails転送パスではdevice_tokenを発行する", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if !hasDeviceTokenCookie(rr) {
			t.Error("Rails転送パスでdevice_token Cookieが発行されなかった")
		}
	})
}

func TestContainsMethod(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		methods  []string
		method   string
		expected bool
	}{
		{"完全一致 (GET)", []string{http.MethodGet}, http.MethodGet, true},
		{"不一致 (GET vs POST)", []string{http.MethodGet}, http.MethodPost, false},
		// POSTはMethod Override前のためPATCHパターンにマッチする
		{"POSTはPATCHにマッチ (Method Override)", []string{http.MethodPatch}, http.MethodPost, true},
		{"POSTはDELETEにマッチ (Method Override)", []string{http.MethodDelete}, http.MethodPost, true},
		// GETはMethod Overrideの対象外なのでPATCHにはマッチしない
		{"GETはPATCHにマッチしない", []string{http.MethodPatch}, http.MethodGet, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := containsMethod(tc.methods, tc.method); got != tc.expected {
				t.Errorf("containsMethod(%v, %q) = %v, want %v", tc.methods, tc.method, got, tc.expected)
			}
		})
	}
}

func TestReverseProxyMiddleware_getFeatureFlagForRequest(t *testing.T) {
	// This test overwrites the global featureFlaggedPatterns, so it does not use t.Parallel().
	// [Ja] このテストはグローバル変数 featureFlaggedPatterns を上書きするため、t.Parallel() を使わない。

	cfg := &config.Config{Domain: "mewst-test.com"}
	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	// featureFlaggedPatterns is empty in production, so inject a test pattern here.
	// [Ja] featureFlaggedPatterns は本番では空のため、テスト用パターンを注入する。
	originalPatterns := featureFlaggedPatterns
	featureFlaggedPatterns = []featureFlaggedPattern{
		{pattern: regexp.MustCompile(`^/@[^/]+$`), flag: model.FeatureFlagExample, methods: []string{http.MethodGet}},
	}
	defer func() { featureFlaggedPatterns = originalPatterns }()

	testCases := []struct {
		name     string
		method   string
		path     string
		expected model.FeatureFlagName
	}{
		{"マッチするパス (プロフィール表示)", http.MethodGet, "/@username", model.FeatureFlagExample},
		{"サブパスはマッチしない (末尾 $)", http.MethodGet, "/@username/posts", ""},
		{"メソッドフィルタによりPOSTはマッチしない", http.MethodPost, "/@username", ""},
		{"マッチしないパス", http.MethodGet, "/settings", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if got := m.getFeatureFlagForRequest(req); got != tc.expected {
				t.Errorf("getFeatureFlagForRequest(%s %q) = %q, want %q", tc.method, tc.path, got, tc.expected)
			}
		})
	}
}

func TestReverseProxyMiddleware_Middleware_FeatureFlag(t *testing.T) {
	// This test overwrites the global featureFlaggedPatterns, so it does not use t.Parallel().
	// [Ja] このテストはグローバル変数 featureFlaggedPatterns を上書きするため、t.Parallel() を使わない。

	_, tx := testutil.SetupTx(t)

	// Per-actor flag: build User -> Profile -> Actor -> Session and attach the flag to the actor resolved via the session token.
	// [Ja] actor 単位フラグ: User → Profile → Actor → Session を作成し、セッショントークン経由で解決した actor にフラグを紐付ける。
	actorUserID := testutil.NewUserBuilder(t, tx).WithEmail("ff-mw-actor@example.com").Build()
	actorProfileID := testutil.NewProfileBuilder(t, tx).WithAtname("ffmwactor").Build()
	actorID := testutil.NewActorBuilder(t, tx).WithUserID(actorUserID).WithProfileID(actorProfileID).Build()
	actorSessionToken := "ff-mw-actor-session-token"
	_ = testutil.NewSessionBuilder(t, tx).WithActorID(actorID).WithToken(actorSessionToken).Build()
	_ = testutil.NewFeatureFlagBuilder(t, tx).WithActorID(actorID).WithName(model.FeatureFlagExample).Build()

	// Per-device flag: the case where a device_token holds the flag.
	// [Ja] device 単位フラグ: device_token がフラグを持つケース。
	deviceToken := "ff-mw-device-token"
	_ = testutil.NewFeatureFlagBuilder(t, tx).WithDeviceToken(deviceToken).WithName(model.FeatureFlagExample).Build()

	// Session of another actor that does not hold the flag; used to verify a viewer for whom the flag is disabled.
	// [Ja] フラグを持たない別 actor のセッション。フラグが無効な閲覧者の検証に使う。
	otherUserID := testutil.NewUserBuilder(t, tx).WithEmail("ff-mw-other@example.com").Build()
	otherProfileID := testutil.NewProfileBuilder(t, tx).WithAtname("ffmwother").Build()
	otherActorID := testutil.NewActorBuilder(t, tx).WithUserID(otherUserID).WithProfileID(otherProfileID).Build()
	otherSessionToken := "ff-mw-other-session-token"
	_ = testutil.NewSessionBuilder(t, tx).WithActorID(otherActorID).WithToken(otherSessionToken).Build()

	originalPatterns := featureFlaggedPatterns
	featureFlaggedPatterns = []featureFlaggedPattern{
		{pattern: regexp.MustCompile(`^/@[^/]+$`), flag: model.FeatureFlagExample, methods: []string{http.MethodGet}},
	}
	defer func() { featureFlaggedPatterns = originalPatterns }()

	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rails-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	cfg := &config.Config{Domain: "mewst-test.com"}

	featureFlagRepo := repository.NewFeatureFlagRepository(testutil.QueriesWithTx(tx))
	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, featureFlagRepo)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Go-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})
	handler := m.Middleware(goHandler)

	t.Run("actorフラグが有効なセッションはGo版で処理される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/@anyone", nil)
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: actorSessionToken})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Go-Handled") != "true" {
			t.Error("フラグが有効なセッションのリクエストがGo版で処理されなかった")
		}
	})

	t.Run("device_tokenフラグが有効なデバイスはGo版で処理される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/@anyone", nil)
		req.AddCookie(&http.Cookie{Name: DeviceTokenCookieName, Value: deviceToken})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Go-Handled") != "true" {
			t.Error("フラグが有効なdevice_tokenのリクエストがGo版で処理されなかった")
		}
	})

	t.Run("フラグが無効なセッションはRails版に転送される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/@anyone", nil)
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: otherSessionToken})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Rails-Handled") != "true" {
			t.Error("フラグが無効なセッションのリクエストがRails版に転送されなかった")
		}
	})

	t.Run("フラグが無効なdevice_tokenはRails版に転送される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/@anyone", nil)
		req.AddCookie(&http.Cookie{Name: DeviceTokenCookieName, Value: "unknown-device-token"})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Rails-Handled") != "true" {
			t.Error("フラグが無効なdevice_tokenのリクエストがRails版に転送されなかった")
		}
	})

	t.Run("どちらのCookieもない場合はRails版に転送される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/@anyone", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Rails-Handled") != "true" {
			t.Error("CookieがないリクエストがRails版に転送されなかった")
		}
	})
}

func TestReverseProxyMiddleware_Middleware_FeatureFlag_ErrorFallback(t *testing.T) {
	// This test overwrites the global featureFlaggedPatterns, so it does not use t.Parallel().
	// [Ja] このテストはグローバル変数 featureFlaggedPatterns を上書きするため、t.Parallel() を使わない。

	originalPatterns := featureFlaggedPatterns
	featureFlaggedPatterns = []featureFlaggedPattern{
		{pattern: regexp.MustCompile(`^/@[^/]+$`), flag: model.FeatureFlagExample, methods: []string{http.MethodGet}},
	}
	defer func() { featureFlaggedPatterns = originalPatterns }()

	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rails-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	cfg := &config.Config{Domain: "mewst-test.com"}

	// Inject a stub that returns an error from the flag check, and verify the request falls back to Rails instead of failing.
	// [Ja] フラグ判定でエラーを返す stub を注入し、サービス断にせず Rails 版へフォールバックすることを検証する。
	repo := stubFeatureFlagChecker{err: errors.New("db error")}
	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, repo)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("判定エラー時はRails版に転送されるべき")
	}))

	req := httptest.NewRequest(http.MethodGet, "/@anyone", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "some-token"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Rails-Handled") != "true" {
		t.Error("判定エラー時のリクエストがRails版に転送されなかった")
	}
}

// TestReverseProxyMiddleware_isGoHandledPattern_NewPost verifies the real
// (production) goHandledPatterns registration for the new-post feature: GET
// /new, POST /posts, GET /links/new and POST /links are always handled by Go,
// while the method/sub-path cases (including the Rails-owned GET /posts/:id) do
// not match and fall through to Rails.
//
// [Ja] 新規投稿機能の実 (本番) goHandledPatterns 登録を検証する。
// GET /new・POST /posts・GET /links/new・POST /links は常に Go 版で処理され、
// メソッド不一致・サブパス (Rails に残す GET /posts/:id を含む) のケースは
// 一致せず Rails にフォールバックすることを確認する。
func TestReverseProxyMiddleware_isGoHandledPattern_NewPost(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Domain: "mewst-test.com"}
	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	testCases := []struct {
		name     string
		method   string
		path     string
		expected bool
	}{
		{"GET /new は Go 処理対象", http.MethodGet, "/new", true},
		{"POST /new はメソッド不一致で対象外", http.MethodPost, "/new", false},
		{"サブパスはマッチしない (末尾 $)", http.MethodGet, "/new/foo", false},
		{"POST /posts は Go 処理対象", http.MethodPost, "/posts", true},
		{"GET /posts はメソッド不一致で対象外", http.MethodGet, "/posts", false},
		// The "^/posts$" anchor must not match the Rails-owned GET /posts/:id.
		// [Ja] "^/posts$" のアンカーは Rails に残す GET /posts/:id にマッチしてはならない。
		{"投稿詳細 /posts/:id はマッチしない (末尾 $)", http.MethodGet, "/posts/123", false},
		{"GET /links/new は Go 処理対象", http.MethodGet, "/links/new", true},
		{"POST /links/new はメソッド不一致で対象外", http.MethodPost, "/links/new", false},
		{"POST /links は Go 処理対象", http.MethodPost, "/links", true},
		{"GET /links はメソッド不一致で対象外", http.MethodGet, "/links", false},
		{"サブパス /links/123 はマッチしない (末尾 $)", http.MethodPost, "/links/123", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.method, tc.path, nil)
			if got := m.isGoHandledPattern(req); got != tc.expected {
				t.Errorf("isGoHandledPattern(%s %q) = %v, want %v", tc.method, tc.path, got, tc.expected)
			}
		})
	}
}

// TestReverseProxyMiddleware_Middleware_NewPostUnconditional verifies the
// end-to-end routing of the new-post endpoints (GET /new, POST /posts, GET
// /links/new, POST /links) through the real goHandledPatterns entries: the Go
// handler always serves them regardless of cookies, while a method mismatch
// (and the Rails-owned GET /posts/:id) falls through to Rails.
//
// [Ja] 新規投稿系エンドポイント (GET /new・POST /posts・GET /links/new・
// POST /links) が実際の goHandledPatterns エントリを通じて振り分けられることを
// E2E で検証する。Cookie の有無に依らず常に Go 版で処理し、メソッド不一致
// (および Rails に残す GET /posts/:id) は Rails に抜けることを確認する。
func TestReverseProxyMiddleware_Middleware_NewPostUnconditional(t *testing.T) {
	// The method-mismatch cases run the full middleware, which reads the
	// featureFlaggedPatterns global. Skip t.Parallel() so this does not race the
	// tests that overwrite that global.
	//
	// [Ja] メソッド不一致のケースはミドルウェア全体を通り featureFlaggedPatterns
	// グローバルを読むため、それを上書きする他テストと競合しないよう t.Parallel()
	// は使わない。

	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rails-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	cfg := &config.Config{Domain: "mewst-test.com"}

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Go-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})
	handler := m.Middleware(goHandler)

	// Go-handled cases: matched exactly by path + method, served by Go regardless of cookies.
	// [Ja] Go 処理対象: パス + メソッドで完全一致し、Cookie の有無に依らず Go 版で処理される。
	goCases := []struct {
		name   string
		method string
		path   string
	}{
		{"GET /new は Go 版で処理される", http.MethodGet, "/new"},
		{"POST /posts は Go 版で処理される", http.MethodPost, "/posts"},
		{"GET /links/new は Go 版で処理される", http.MethodGet, "/links/new"},
		{"POST /links は Go 版で処理される", http.MethodPost, "/links"},
	}
	for _, tc := range goCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Header().Get("X-Go-Handled") != "true" {
				t.Errorf("%s %s が Go 版で処理されなかった", tc.method, tc.path)
			}
		})
	}

	// Rails-proxied cases: method mismatch or sub-path, so they fall through to Rails.
	// [Ja] Rails プロキシ対象: メソッド不一致やサブパスのため Rails にフォールバックする。
	railsCases := []struct {
		name   string
		method string
		path   string
	}{
		{"POST /new はメソッド不一致で Rails 版に転送される", http.MethodPost, "/new"},
		{"GET /posts はメソッド不一致で Rails 版に転送される", http.MethodGet, "/posts"},
		// The "^/posts$" anchor leaves the Rails-owned GET /posts/:id with Rails.
		// [Ja] "^/posts$" のアンカーにより Rails に残す GET /posts/:id は Rails に残る。
		{"投稿詳細 GET /posts/:id は Rails 版に転送される", http.MethodGet, "/posts/123"},
		{"GET /links はメソッド不一致で Rails 版に転送される", http.MethodGet, "/links"},
	}
	for _, tc := range railsCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Header().Get("X-Rails-Handled") != "true" {
				t.Errorf("%s %s が Rails 版に転送されなかった", tc.method, tc.path)
			}
		})
	}
}

// TestReverseProxyMiddleware_isGoHandledPattern_Settings verifies the real
// (production) goHandledPatterns registration for /settings: GET /settings is
// always handled by Go, while the Rails-owned sub-pages (GET /settings/profile,
// /settings/user, /settings/email) and a method mismatch do not match and fall
// through to Rails.
//
// [Ja] 設定画面の実 (本番) goHandledPatterns 登録を検証する。
// GET /settings は常に Go 版で処理され、Rails に残すサブページ
// (GET /settings/profile・/settings/user・/settings/email) とメソッド不一致は
// 一致せず Rails にフォールバックすることを確認する。
func TestReverseProxyMiddleware_isGoHandledPattern_Settings(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Domain: "mewst-test.com"}
	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	testCases := []struct {
		name     string
		method   string
		path     string
		expected bool
	}{
		{"GET /settings は Go 処理対象", http.MethodGet, "/settings", true},
		{"POST /settings はメソッド不一致で対象外", http.MethodPost, "/settings", false},
		// The "^/settings$" anchor must not match the Rails-owned sub-pages.
		//
		// [Ja] "^/settings$" のアンカーは Rails に残すサブページにマッチしてはならない。
		{"プロフィール編集 /settings/profile はマッチしない (末尾 $)", http.MethodGet, "/settings/profile", false},
		{"ユーザー編集 /settings/user はマッチしない (末尾 $)", http.MethodGet, "/settings/user", false},
		{"メール変更 /settings/email はマッチしない (末尾 $)", http.MethodGet, "/settings/email", false},
		{"末尾スラッシュ /settings/ はマッチしない (末尾 $)", http.MethodGet, "/settings/", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.method, tc.path, nil)
			if got := m.isGoHandledPattern(req); got != tc.expected {
				t.Errorf("isGoHandledPattern(%s %q) = %v, want %v", tc.method, tc.path, got, tc.expected)
			}
		})
	}
}

// TestReverseProxyMiddleware_Middleware_Settings verifies the end-to-end routing
// of /settings through the real goHandledPatterns entry: the Go handler serves
// GET /settings regardless of cookies, while the Rails-owned sub-pages
// (/settings/profile, /settings/user, /settings/email) and a method mismatch
// (POST /settings) fall through to Rails.
//
// [Ja] 設定画面が実際の goHandledPatterns エントリを通じて振り分けられることを
// E2E で検証する。Cookie の有無に依らず GET /settings は Go 版で処理し、
// Rails に残すサブページ (/settings/profile・/settings/user・/settings/email) と
// メソッド不一致 (POST /settings) は Rails に抜けることを確認する。
func TestReverseProxyMiddleware_Middleware_Settings(t *testing.T) {
	// The Rails fall-through cases (the sub-pages and the method mismatch) run the
	// full middleware, which reads the featureFlaggedPatterns global. Skip
	// t.Parallel() so this does not race the tests that overwrite that global.
	//
	// [Ja] Rails にフォールバックする各ケース (サブページとメソッド不一致) は
	// ミドルウェア全体を通り featureFlaggedPatterns グローバルを読むため、それを
	// 上書きする他テストと競合しないよう t.Parallel() は使わない。

	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rails-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	cfg := &config.Config{Domain: "mewst-test.com"}

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Go-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})
	handler := m.Middleware(goHandler)

	t.Run("GET /settings は Go 版で処理される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Go-Handled") != "true" {
			t.Error("GET /settings が Go 版で処理されなかった")
		}
	})

	// Sub-pages and a method mismatch fall through to Rails.
	//
	// [Ja] サブページとメソッド不一致は Rails にフォールバックする。
	railsCases := []struct {
		name   string
		method string
		path   string
	}{
		{"GET /settings/profile は Rails 版に転送される", http.MethodGet, "/settings/profile"},
		{"GET /settings/user は Rails 版に転送される", http.MethodGet, "/settings/user"},
		{"GET /settings/email は Rails 版に転送される", http.MethodGet, "/settings/email"},
		{"POST /settings はメソッド不一致で Rails 版に転送される", http.MethodPost, "/settings"},
	}
	for _, tc := range railsCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Header().Get("X-Rails-Handled") != "true" {
				t.Errorf("%s %s が Rails 版に転送されなかった", tc.method, tc.path)
			}
		})
	}
}

func TestReverseProxyMiddleware_Middleware_FeatureFlag_NilRepo(t *testing.T) {
	// This test overwrites the global featureFlaggedPatterns, so it does not use t.Parallel().
	// [Ja] このテストはグローバル変数 featureFlaggedPatterns を上書きするため、t.Parallel() を使わない。

	originalPatterns := featureFlaggedPatterns
	featureFlaggedPatterns = []featureFlaggedPattern{
		{pattern: regexp.MustCompile(`^/@[^/]+$`), flag: model.FeatureFlagExample, methods: []string{http.MethodGet}},
	}
	defer func() { featureFlaggedPatterns = originalPatterns }()

	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rails-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	cfg := &config.Config{Domain: "mewst-test.com"}

	// When featureFlagRepo is nil the flag check is skipped, so even a matching pattern is forwarded to Rails.
	// [Ja] featureFlagRepo が nil のときはフラグ判定をスキップするため、パターンにマッチしても Rails 版へ転送される。
	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("featureFlagRepoがnilの場合はRails版に転送されるべき")
	}))

	req := httptest.NewRequest(http.MethodGet, "/@anyone", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "some-token"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Rails-Handled") != "true" {
		t.Error("featureFlagRepoがnilのリクエストがRails版に転送されなかった")
	}
}
