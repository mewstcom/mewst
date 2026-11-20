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

// stubFeatureFlagCheckerはテスト用に固定値を返すfeatureFlagChecker。
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
				t.Errorf("ステータスコードが期待と異なる: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			if rr.Body.String() != tc.expectedBody {
				t.Errorf("レスポンスボディが期待と異なる: 実測値 = %q、期待値 = %q", rr.Body.String(), tc.expectedBody)
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
			t.Errorf("X-Forwarded-Protoが設定されていない: 実測値 = %q", r.Header.Get("X-Forwarded-Proto"))
		}
		if r.Header.Get("X-Forwarded-Host") != "mewst-test.com" {
			t.Errorf("X-Forwarded-Hostが設定されていない: 実測値 = %q", r.Header.Get("X-Forwarded-Host"))
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
		// "/settings" 本体はGo版で処理されるようになった (後述のSettingsテストを参照)。
		// サブページはRailsに残るため、設定系の代表としてサブページを使う。
		{"設定サブページ", "/settings/profile", "Rails response"},
		{"投稿ページ", "/posts/123", "Rails response"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコードが期待と異なる: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			if rr.Body.String() != tc.expectedBody {
				t.Errorf("レスポンスボディが期待と異なる: 実測値 = %q、期待値 = %q", rr.Body.String(), tc.expectedBody)
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
				t.Errorf("isGoHandledPath(%q) = %v、期待値 = %v", tc.path, actual, tc.expected)
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
		t.Errorf("ステータスコードが期待と異なる: 実測値 = %v、期待値 = %v", rr.Code, http.StatusBadGateway)
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
				t.Errorf("ヘッダー%sが期待と異なる: 実測値 = %q、期待値 = %q", name, actual, expected)
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
		t.Errorf("ステータスコードが期待と異なる: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
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

	// 様々なHTTPメソッドがRails版にプロキシされることを確認する。
	// "/settings/profile" はgoHandledPathsにもgoHandledPatternsにも含まれない
	// 純粋なRailsサブパスで、どのHTTPメソッドでもRailsにプロキシされる。
	// (親のGET /settingsはGoの処理対象になり、POST /postsもGoの処理対象のため、
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
				t.Errorf("ステータスコードが期待と異なる: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			if rr.Body.String() != tc.expectedBody {
				t.Errorf("レスポンスボディが期待と異なる: 実測値 = %q、期待値 = %q", rr.Body.String(), tc.expectedBody)
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
			t.Errorf("SameSite = %v、期待値 = %v", deviceCookie.SameSite, http.SameSiteLaxMode)
		}
		if deviceCookie.Domain != "mewst-test.com" {
			t.Errorf("Domain = %q、期待値 = %q", deviceCookie.Domain, "mewst-test.com")
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

// device_token CookieがGo処理パスの判定より後で発行され、静的アセットや
// ヘルスチェックにはSet-Cookieが付かないことを検証する。
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
				t.Errorf("containsMethod(%v, %q) = %v、期待値 = %v", tc.methods, tc.method, got, tc.expected)
			}
		})
	}
}

func TestReverseProxyMiddleware_getFeatureFlagForRequest(t *testing.T) {
	// このテストはグローバル変数featureFlaggedPatternsを上書きするため、t.Parallel() を使わない。

	cfg := &config.Config{Domain: "mewst-test.com"}
	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)
	if err != nil {
		t.Fatalf("ミドルウェアの作成に失敗: %v", err)
	}

	// featureFlaggedPatternsを固定のテスト用パターンで上書きし、本番の
	// 登録内容に依存しないようにする。
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
				t.Errorf("getFeatureFlagForRequest(%s %q) = %q、期待値 = %q", tc.method, tc.path, got, tc.expected)
			}
		})
	}
}

func TestReverseProxyMiddleware_Middleware_FeatureFlag(t *testing.T) {
	// このテストはグローバル変数featureFlaggedPatternsを上書きするため、t.Parallel() を使わない。

	_, tx := testutil.SetupTx(t)

	// actor単位フラグ: User → Profile → Actor → Sessionを作成し、セッショントークン経由で解決したactorにフラグを紐付ける。
	actorUserID := testutil.NewUserBuilder(t, tx).WithEmail("ff-mw-actor@example.com").Build()
	actorProfileID := testutil.NewProfileBuilder(t, tx).WithAtname("ffmwactor").Build()
	actorID := testutil.NewActorBuilder(t, tx).WithUserID(actorUserID).WithProfileID(actorProfileID).Build()
	actorSessionToken := "ff-mw-actor-session-token"
	_ = testutil.NewSessionBuilder(t, tx).WithActorID(actorID).WithToken(actorSessionToken).Build()
	_ = testutil.NewFeatureFlagBuilder(t, tx).WithActorID(actorID).WithName(model.FeatureFlagExample).Build()

	// device単位フラグ: device_tokenがフラグを持つケース。
	deviceToken := "ff-mw-device-token"
	_ = testutil.NewFeatureFlagBuilder(t, tx).WithDeviceToken(deviceToken).WithName(model.FeatureFlagExample).Build()

	// フラグを持たない別actorのセッション。フラグが無効な閲覧者の検証に使う。
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
	// このテストはグローバル変数featureFlaggedPatternsを上書きするため、t.Parallel() を使わない。

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

	// フラグ判定でエラーを返すstubを注入し、サービス断にせずRails版へフォールバックすることを検証する。
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

// 新規投稿機能の実 (本番) goHandledPatterns登録を検証する。
// GET /new・POST /posts・GET /links/new・POST /linksは常にGo版で処理され、
// メソッド不一致・サブパス (Railsに残すGET /posts/:idを含む) のケースは
// 一致せずRailsにフォールバックすることを確認する。
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
		// "^/posts$" のアンカーはRailsに残すGET /posts/:idにマッチしてはならない。
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
				t.Errorf("isGoHandledPattern(%s %q) = %v、期待値 = %v", tc.method, tc.path, got, tc.expected)
			}
		})
	}
}

// 新規投稿系エンドポイント (GET /new・POST /posts・GET /links/new・
// POST /links) が実際のgoHandledPatternsエントリを通じて振り分けられることを
// E2Eで検証する。Cookieの有無に依らず常にGo版で処理し、メソッド不一致
// (およびRailsに残すGET /posts/:id) はRailsに抜けることを確認する。
func TestReverseProxyMiddleware_Middleware_NewPostUnconditional(t *testing.T) {
	// メソッド不一致のケースはミドルウェア全体を通りfeatureFlaggedPatterns
	// グローバルを読むため、それを上書きする他テストと競合しないようt.Parallel()
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

	// Go処理対象: パス + メソッドで完全一致し、Cookieの有無に依らずGo版で処理される。
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
				t.Errorf("%s %sがGo版で処理されなかった", tc.method, tc.path)
			}
		})
	}

	// Railsプロキシ対象: メソッド不一致やサブパスのためRailsにフォールバックする。
	railsCases := []struct {
		name   string
		method string
		path   string
	}{
		{"POST /new はメソッド不一致で Rails 版に転送される", http.MethodPost, "/new"},
		{"GET /posts はメソッド不一致で Rails 版に転送される", http.MethodGet, "/posts"},
		// "^/posts$" のアンカーによりRailsに残すGET /posts/:idはRailsに残る。
		{"投稿詳細 GET /posts/:id は Rails 版に転送される", http.MethodGet, "/posts/123"},
		{"GET /links はメソッド不一致で Rails 版に転送される", http.MethodGet, "/links"},
	}
	for _, tc := range railsCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Header().Get("X-Rails-Handled") != "true" {
				t.Errorf("%s %sがRails版に転送されなかった", tc.method, tc.path)
			}
		})
	}
}

// 設定画面の実 (本番) goHandledPatterns登録を検証する。
// GET /settingsは常にGo版で処理され、Railsに残すサブページ
// (GET /settings/profile・/settings/user・/settings/email) とメソッド不一致は
// 一致せずRailsにフォールバックすることを確認する。
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
		// "^/settings$" のアンカーはRailsに残すサブページにマッチしてはならない。
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
				t.Errorf("isGoHandledPattern(%s %q) = %v、期待値 = %v", tc.method, tc.path, got, tc.expected)
			}
		})
	}
}

// 設定画面が実際のgoHandledPatternsエントリを通じて振り分けられることを
// E2Eで検証する。Cookieの有無に依らずGET /settingsはGo版で処理し、
// Railsに残すサブページ (/settings/profile・/settings/user・/settings/email) と
// メソッド不一致 (POST /settings) はRailsに抜けることを確認する。
func TestReverseProxyMiddleware_Middleware_Settings(t *testing.T) {
	// Railsにフォールバックする各ケース (サブページとメソッド不一致) は
	// ミドルウェア全体を通りfeatureFlaggedPatternsグローバルを読むため、それを
	// 上書きする他テストと競合しないようt.Parallel() は使わない。

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

	t.Run("GET /settingsはGo版で処理される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Go-Handled") != "true" {
			t.Error("GET /settingsがGo版で処理されなかった")
		}
	})

	// サブページとメソッド不一致はRailsにフォールバックする。
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
				t.Errorf("%s %sがRails版に転送されなかった", tc.method, tc.path)
			}
		})
	}
}

// エクスポート機能の実 (本番) goHandledPatterns登録を検証する。
// GET/POST /settings/exportとGET /settings/export/downloadは常にGo版で
// 処理され、メソッド不一致と近接するパスは一致せずRailsへのプロキシに進む。
func TestReverseProxyMiddleware_isGoHandledPattern_Export(t *testing.T) {
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
		{"GET /settings/export は Go 処理対象", http.MethodGet, "/settings/export", true},
		{"POST /settings/export は Go 処理対象", http.MethodPost, "/settings/export", true},
		{"GET /settings/export/download は Go 処理対象", http.MethodGet, "/settings/export/download", true},
		{"POST /settings/export/download はメソッド不一致で対象外", http.MethodPost, "/settings/export/download", false},
		{"DELETE /settings/export はメソッド不一致で対象外", http.MethodDelete, "/settings/export", false},
		{"サブパス /settings/export/foo はマッチしない (末尾 $)", http.MethodGet, "/settings/export/foo", false},
		{"前方一致 /settings/exports はマッチしない (完全一致)", http.MethodGet, "/settings/exports", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.method, tc.path, nil)
			if got := m.isGoHandledPattern(req); got != tc.expected {
				t.Errorf("isGoHandledPattern(%s %q) = %v、期待値 = %v", tc.method, tc.path, got, tc.expected)
			}
		})
	}
}

// エクスポート系ルートが実際のgoHandledPatternsエントリを通じて振り分け
// られることをE2Eで検証する。フラグでゲートしなくなったためCookieの有無に
// 依らずGo版で処理し、メソッド不一致は引き続きRailsに抜けることを確認する。
func TestReverseProxyMiddleware_Middleware_Export(t *testing.T) {
	// Railsにフォールバックするケースはミドルウェア全体を通り
	// featureFlaggedPatternsグローバルを読むため、それを上書きする他テストと
	// 競合しないようt.Parallel() は使わない。

	_, tx := testutil.SetupTx(t)

	userID := testutil.NewUserBuilder(t, tx).WithEmail("export-mw@example.com").Build()
	profileID := testutil.NewProfileBuilder(t, tx).WithAtname("exportmw").Build()
	actorID := testutil.NewActorBuilder(t, tx).WithUserID(userID).WithProfileID(profileID).Build()
	sessionToken := "export-mw-session-token"
	_ = testutil.NewSessionBuilder(t, tx).WithActorID(actorID).WithToken(sessionToken).Build()

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

	goCases := []struct {
		name   string
		method string
		path   string
		cookie *http.Cookie
	}{
		{"GET /settings/export は Go 版で処理される", http.MethodGet, "/settings/export", &http.Cookie{Name: session.CookieName, Value: sessionToken}},
		{"POST /settings/export は Go 版で処理される", http.MethodPost, "/settings/export", &http.Cookie{Name: session.CookieName, Value: sessionToken}},
		{"GET /settings/export/download は Go 版で処理される", http.MethodGet, "/settings/export/download", &http.Cookie{Name: session.CookieName, Value: sessionToken}},
		{"Cookie なしの GET /settings/export も Go 版で処理される", http.MethodGet, "/settings/export", nil},
	}
	for _, tc := range goCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Header().Get("X-Go-Handled") != "true" {
				t.Errorf("%s %sがGo版で処理されなかった", tc.method, tc.path)
			}
		})
	}

	t.Run("DELETE /settings/exportはメソッド不一致でRails版に転送される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/settings/export", nil)
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sessionToken})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Rails-Handled") != "true" {
			t.Error("DELETE /settings/exportがRails版に転送されなかった")
		}
	})
}

func TestReverseProxyMiddleware_Middleware_FeatureFlag_NilRepo(t *testing.T) {
	// このテストはグローバル変数featureFlaggedPatternsを上書きするため、t.Parallel() を使わない。

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

	// featureFlagRepoがnilのときはフラグ判定をスキップするため、パターンにマッチしてもRails版へ転送される。
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
