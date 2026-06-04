package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/clientip"
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/httperror"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
)

// DeviceTokenCookieName is the cookie key that identifies a browser (device)
// regardless of login state, used for feature-flag targeting.
//
// [Ja] DeviceTokenCookieName はログイン状態に依らずブラウザ (デバイス) を
// 識別する Cookie キー名。フィーチャーフラグの出し分けに使う。
const DeviceTokenCookieName = "device_token"

// featureFlagChecker reports whether a feature flag is enabled for the viewer.
// repository.FeatureFlagRepository satisfies this interface.
//
// [Ja] featureFlagChecker は閲覧者に対してフィーチャーフラグが有効かを返す。
// repository.FeatureFlagRepository がこのインターフェースを満たす。
type featureFlagChecker interface {
	IsEnabledForDevice(ctx context.Context, deviceToken string, sessionToken string, name model.FeatureFlagName) (bool, error)
}

// featureFlaggedPattern defines a URL pattern gated by a feature flag.
// [Ja] featureFlaggedPattern はフィーチャーフラグで制御する URL パターンを定義する。
type featureFlaggedPattern struct {
	pattern *regexp.Regexp
	flag    model.FeatureFlagName
	methods []string // nil or empty matches every method. [Ja] nil または空なら全メソッドにマッチ
}

// featureFlaggedPatterns lists the URL patterns gated by a feature flag.
//
// Each entry anchors its path regexp with "^...$" so it matches the exact path
// and not sub-paths, and pairs it with an HTTP method set and the flag that
// gates it. A matching request is served by Go only when the flag is enabled
// for the viewer; otherwise it falls through to the Rails proxy.
//
// [Ja] featureFlaggedPatterns はフィーチャーフラグで制御する URL パターンの一覧。
//
// 各エントリはパスの正規表現を "^...$" でアンカーしてサブパスではなく完全一致
// させ、HTTP メソッドの集合とそれをゲートするフラグを対応付ける。一致した
// リクエストは閲覧者にフラグが有効なときだけ Go 版で処理し、無効なら Rails への
// プロキシに進む。
var featureFlaggedPatterns = []featureFlaggedPattern{
	// GET /new: the Go new-post form. The "^/new$" anchor keeps it from matching
	// sub-paths.
	//
	// [Ja] GET /new: Go 版の新規投稿フォーム。"^/new$" のアンカーでサブパスには
	// マッチさせない。
	{
		pattern: regexp.MustCompile(`^/new$`),
		flag:    model.FeatureFlagNewPost,
		methods: []string{http.MethodGet},
	},
	// POST /posts: the Go post-creation endpoint, gated under the same flag as
	// /new. The "^/posts$" anchor matches only the exact path, so the Rails-owned
	// GET /posts/:id (post detail, out of scope) is not swept in.
	//
	// [Ja] POST /posts: Go 版の投稿作成エンドポイント。/new と同じフラグの配下に置く。
	// "^/posts$" のアンカーで完全一致のみにマッチさせるため、Rails に残す
	// GET /posts/:id (投稿詳細・スコープ外) を巻き込まない。
	{
		pattern: regexp.MustCompile(`^/posts$`),
		flag:    model.FeatureFlagNewPost,
		methods: []string{http.MethodPost},
	},
}

// ReverseProxyMiddleware is the reverse-proxy middleware to the Rails version.
// [Ja] ReverseProxyMiddleware は Rails 版へのリバースプロキシミドルウェア。
type ReverseProxyMiddleware struct {
	railsURL        *url.URL
	proxy           *httputil.ReverseProxy
	cfg             *config.Config
	featureFlagRepo featureFlagChecker
}

// Go版で処理するパス (ホワイトリスト)
// これらのパスはRails版にプロキシせず、Go版のハンドラーで処理する
var goHandledPaths = []string{
	"/static",             // 静的ファイル (CSS、JS、画像など)
	"/health",             // ヘルスチェックエンドポイント
	"/manifest.json",      // Web App Manifest
	"/sign_in",            // ログインページ・処理
	"/sign_up",            // サインアップページ・処理
	"/sign_out",           // ログアウト処理
	"/password_reset",     // パスワードリセット開始
	"/email_confirmation", // メール確認
	"/password",           // パスワード更新
	"/accounts",           // アカウント作成
}

// NewReverseProxyMiddleware creates a new ReverseProxyMiddleware.
// When featureFlagRepo is nil, feature-flag checks are skipped.
//
// [Ja] NewReverseProxyMiddleware は新しい ReverseProxyMiddleware を作成する。
// featureFlagRepo が nil の場合、フィーチャーフラグ判定はスキップされる。
func NewReverseProxyMiddleware(railsURL string, cfg *config.Config, featureFlagRepo featureFlagChecker) (*ReverseProxyMiddleware, error) {
	parsedURL, err := url.Parse(railsURL)
	if err != nil {
		return nil, err
	}

	// httputil.ReverseProxyを作成
	proxy := &httputil.ReverseProxy{}

	// カスタムのHTTP Transportを設定 (タイムアウトと接続プーリング)
	proxy.Transport = &http.Transport{
		// 接続タイムアウト: 10秒
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// レスポンスヘッダー読み取りタイムアウト: 30秒
		ResponseHeaderTimeout: 30 * time.Second,
		// 接続プーリングの設定
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Customize header rewriting via the proxy's Rewrite function. ReverseProxy strips Forwarded /
	// X-Forwarded-For / X-Forwarded-Host / X-Forwarded-Proto from Out.Header before calling Rewrite,
	// so read the original values from pr.In.Header when they need to be preserved.
	//
	// [Ja] プロキシの Rewrite 関数でヘッダー設定を行う。httputil.ReverseProxy は Rewrite 呼び出し前に
	// Forwarded / X-Forwarded-For / X-Forwarded-Host / X-Forwarded-Proto を Out.Header から削除するため、
	// 元の値を参照したい場合は pr.In.Header から取得する必要がある。
	proxy.Rewrite = func(pr *httputil.ProxyRequest) {
		// Rewrite the URL to the Rails host. SetURL sets Out.Host = "", so follow it with
		// Out.Host = In.Host to keep forwarding the client's Host header to Rails unchanged.
		//
		// [Ja] URL を Rails 版のホストに書き換える。SetURL は Out.Host = "" をセットしてしまうため、
		// 続けて Out.Host = In.Host を設定し、クライアントが送ってきた Host ヘッダをそのまま Rails 版に
		// 転送する挙動を維持する。
		pr.SetURL(parsedURL)
		pr.Out.Host = pr.In.Host

		// クライアントIPアドレスを取得
		clientIP := clientip.GetClientIP(pr.In)

		// X-Forwarded-Forヘッダーの設定
		if originalXForwardedFor := pr.In.Header.Get("X-Forwarded-For"); originalXForwardedFor != "" {
			// 既存の値を維持 (Cloudflareなどが設定した値を保持)
			pr.Out.Header.Set("X-Forwarded-For", originalXForwardedFor)
		} else {
			// 既存の値がない場合、clientIPを設定
			pr.Out.Header.Set("X-Forwarded-For", clientIP)
		}

		// X-Real-IPヘッダーの設定 (既存の値がない場合のみ)
		if originalXRealIP := pr.In.Header.Get("X-Real-IP"); originalXRealIP != "" {
			pr.Out.Header.Set("X-Real-IP", originalXRealIP)
		} else {
			pr.Out.Header.Set("X-Real-IP", clientIP)
		}

		// X-Forwarded-Protoの設定
		pr.Out.Header.Set("X-Forwarded-Proto", "https")

		// X-Forwarded-Hostの設定
		pr.Out.Header.Set("X-Forwarded-Host", cfg.Domain)

		// ログ出力 (開発者向け)
		slog.Info("リバースプロキシでRails版にリクエストを転送",
			"path", pr.In.URL.Path,
			"method", pr.In.Method,
			"target", parsedURL.String()+pr.In.URL.Path,
			"client_ip", clientIP,
		)
	}

	// レスポンス処理後のログ出力 (成功時)
	proxy.ModifyResponse = func(resp *http.Response) error {
		// プロキシが成功した場合のレスポンスログを出力 (開発者向け)
		slog.Info("Rails版からレスポンスを受信",
			"status_code", resp.StatusCode,
			"status", resp.Status,
			"path", resp.Request.URL.Path,
			"method", resp.Request.Method,
		)
		return nil
	}

	// エラーハンドラーをカスタマイズ
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		ctx := r.Context()

		// Rails 版が 502 を返したり接続が切れた場合は Go 版の障害ではなく
		// Rails 版の外部要因 (= Go 版の Sentry に Issue を作るほどではない) のため Warn に留める。
		// slog handler は LevelError 以上のみ Sentry に送るので、ここを Warn にすることで
		// Sentry へのノイズ送信を構造的に防止する。
		slog.WarnContext(ctx, "Rails版へのプロキシでエラーが発生",
			"error", err,
			"path", r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
		)

		// reverse_proxy は i18n.Middleware より前に動くため、ロケールを検出して context に載せ替えてから httperror に委譲する
		locale := i18n.DetectLanguage(r)
		ctx = i18n.SetLocale(ctx, locale)
		httperror.BadGateway(w, r.WithContext(ctx))
	}

	return &ReverseProxyMiddleware{
		railsURL:        parsedURL,
		proxy:           proxy,
		cfg:             cfg,
		featureFlagRepo: featureFlagRepo,
	}, nil
}

// Middleware returns the HTTP middleware.
// [Ja] Middleware は HTTP ミドルウェアを返す。
func (m *ReverseProxyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Paths always handled by Go (whitelist).
		// [Ja] 1. 常に Go 版で処理するパス (ホワイトリスト)。
		if m.isGoHandledPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Issue a device_token cookie when one is absent, so feature-flag
		// decisions can identify the browser regardless of login state. This
		// runs after the Go-handled check so static assets and health checks
		// (which never need feature-flag targeting) don't emit a Set-Cookie.
		//
		// [Ja] device_token Cookie が無ければ発行する。ログイン状態に依らず
		// ブラウザを識別してフィーチャーフラグ判定に使えるようにするため。
		// 静的アセットやヘルスチェック (フィーチャーフラグの出し分けが不要なパス) に
		// Set-Cookie を出さないよう、Go 処理パスの判定より後で発行する。
		m.ensureDeviceToken(w, r)

		// 2. Feature-flagged paths: handle in Go only when the flag is enabled
		// for this viewer; otherwise fall through to the Rails proxy.
		//
		// [Ja] 2. フィーチャーフラグで制御するパス。閲覧者に対してフラグが
		// 有効なときだけ Go 版で処理し、無効なら Rails へのプロキシに進む。
		if flagName := m.getFeatureFlagForRequest(r); flagName != "" {
			if m.isFeatureFlagEnabled(r, flagName) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// 3. Everything else proxies to Rails.
		// [Ja] 3. それ以外はすべて Rails にプロキシする。
		m.proxy.ServeHTTP(w, r)
	})
}

// ensureDeviceToken issues a device_token cookie when the request has none.
// [Ja] ensureDeviceToken はリクエストに device_token Cookie が無ければ発行する。
func (m *ReverseProxyMiddleware) ensureDeviceToken(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(DeviceTokenCookieName); err == nil {
		return // Cookie already exists. [Ja] 既に Cookie が存在する
	}

	token, err := auth.GenerateSecureToken()
	if err != nil {
		slog.WarnContext(r.Context(), "device_tokenの生成に失敗", "error", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     DeviceTokenCookieName,
		Value:    token,
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		MaxAge:   10 * 365 * 24 * 60 * 60, // 10 years. [Ja] 10年
		HttpOnly: true,
		Secure:   m.cfg.SessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// isGoHandledPath はGo版で処理するパスかどうかを判定
func (m *ReverseProxyMiddleware) isGoHandledPath(path string) bool {
	for _, p := range goHandledPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// getFeatureFlagForRequest returns the feature flag name matching the request's
// path and method, or an empty string when no pattern matches.
//
// [Ja] getFeatureFlagForRequest はリクエストのパスとメソッドに一致する
// フィーチャーフラグ名を返す。一致するパターンが無ければ空文字列を返す。
func (m *ReverseProxyMiddleware) getFeatureFlagForRequest(r *http.Request) model.FeatureFlagName {
	for _, fp := range featureFlaggedPatterns {
		if !fp.pattern.MatchString(r.URL.Path) {
			continue
		}
		if len(fp.methods) > 0 && !containsMethod(fp.methods, r.Method) {
			continue
		}
		return fp.flag
	}
	return ""
}

// containsMethod reports whether method is contained in methods.
//
// HTML forms support only GET and POST, so PATCH/PUT/DELETE requests are sent
// as POST plus a _method parameter (the Method Override pattern). This
// middleware runs before the Method Override middleware, so a POST request must
// also match PATCH/PUT/DELETE patterns.
//
// [Ja] containsMethod は method が methods に含まれるかを判定する。
//
// HTML フォームは GET と POST のみをサポートするため、PATCH/PUT/DELETE は
// POST + _method パラメータとして送信される (Method Override パターン)。
// 本ミドルウェアは Method Override ミドルウェアより前に実行されるため、
// POST リクエストも PATCH/PUT/DELETE パターンにマッチさせる必要がある。
func containsMethod(methods []string, method string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}

	// A POST request may be converted to PATCH/PUT/DELETE via Method Override.
	// [Ja] POST リクエストは Method Override 経由で PATCH/PUT/DELETE に変換される可能性がある。
	if method == http.MethodPost {
		for _, m := range methods {
			switch m {
			case http.MethodPatch, http.MethodPut, http.MethodDelete:
				return true
			}
		}
	}

	return false
}

// isFeatureFlagEnabled reports whether the feature flag is enabled for the
// request based on its cookies. It returns false on error or when no identifying
// cookie is present, so the request falls back to the Rails version.
//
// [Ja] isFeatureFlagEnabled はリクエストの Cookie からフィーチャーフラグが
// 有効かどうかを判定する。エラー時または識別用 Cookie 不在時は false を返し、
// Rails 版にフォールバックする。
func (m *ReverseProxyMiddleware) isFeatureFlagEnabled(r *http.Request, flagName model.FeatureFlagName) bool {
	if m.featureFlagRepo == nil {
		return false
	}

	// Read the device_token cookie value.
	// [Ja] device_token Cookie の値を取得する。
	deviceToken := ""
	if cookie, err := r.Cookie(DeviceTokenCookieName); err == nil {
		deviceToken = cookie.Value
	}

	// Read the session token cookie shared with the Rails version.
	// [Ja] Rails 版と共有するセッショントークン Cookie の値を取得する。
	sessionToken := ""
	if cookie, err := r.Cookie(session.CookieName); err == nil {
		sessionToken = cookie.Value
	}

	// Fall back to Rails when neither cookie is present.
	// [Ja] どちらの Cookie も存在しない場合は Rails 版にフォールバックする。
	if deviceToken == "" && sessionToken == "" {
		return false
	}

	// Evaluate device_token and the session-derived actor in a single query.
	// [Ja] device_token とセッション経由の actor を 1 クエリで判定する。
	enabled, err := m.featureFlagRepo.IsEnabledForDevice(r.Context(), deviceToken, sessionToken, flagName)
	if err != nil {
		slog.WarnContext(r.Context(), "フィーチャーフラグ判定でエラーが発生 (Rails版にフォールバック)",
			"error", err,
			"flag", flagName,
			"path", r.URL.Path,
		)
		return false
	}

	return enabled
}
