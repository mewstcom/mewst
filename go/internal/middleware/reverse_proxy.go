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

// DeviceTokenCookieNameはログイン状態に依らずブラウザ (デバイス) を
// 識別するCookieキー名。フィーチャーフラグの出し分けに使う。
const DeviceTokenCookieName = "device_token"

// featureFlagCheckerは閲覧者に対してフィーチャーフラグが有効かを返す。
// repository.FeatureFlagRepositoryがこのインターフェースを満たす。
type featureFlagChecker interface {
	IsEnabledForDevice(ctx context.Context, deviceToken string, sessionToken string, name model.FeatureFlagName) (bool, error)
}

// featureFlaggedPatternはフィーチャーフラグで制御するURLパターンを定義する。
type featureFlaggedPattern struct {
	pattern *regexp.Regexp
	flag    model.FeatureFlagName
	methods []string // nilまたは空なら全メソッドにマッチ
}

// featureFlaggedPatternsはフィーチャーフラグで制御するURLパターンの一覧。
//
// 各エントリはパスの正規表現を "^...$" でアンカーしてサブパスではなく完全一致
// させ、HTTPメソッドの集合とそれをゲートするフラグを対応付ける。一致した
// リクエストは閲覧者にフラグが有効なときだけGo版で処理し、無効ならRailsへの
// プロキシに進む。
//
// フラグの内側にあるルートが無い間、一覧は空になる。公開した機能は自身の
// ルートをgoHandledPatternsへ移し、この一覧は次の機能に譲る。
var featureFlaggedPatterns = []featureFlaggedPattern{}

// ReverseProxyMiddlewareはRails版へのリバースプロキシミドルウェア。
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

// goHandledPatternは常にGo版で処理する完全一致URLパターンとHTTP
// メソッドの集合を定義する。
type goHandledPattern struct {
	pattern *regexp.Regexp
	methods []string // nilまたは空なら全メソッドにマッチ
}

// goHandledPatternsは常にGo版で処理するURLパターンの一覧。完全一致の
// パス正規表現とHTTPメソッドの集合で識別する。
//
// goHandledPathsはprefix一致のため、ここでは粗すぎる: "/posts" を足すとRails
// に残すGET /posts/:id (投稿詳細) まで奪ってしまう。これらのエンドポイントは
// exact-path + methodの精度が要るため、"^...$" のアンカーとメソッド集合により
// POST /postsはPOST /postsだけにマッチさせ、GET /posts/:idはRailsに残す。
var goHandledPatterns = []goHandledPattern{
	// GET /new: 新規投稿フォーム。
	{pattern: regexp.MustCompile(`^/new$`), methods: []string{http.MethodGet}},
	// POST /posts: 投稿作成。"^/posts$" のアンカーでGET /posts/:idはRailsに残す。
	{pattern: regexp.MustCompile(`^/posts$`), methods: []string{http.MethodPost}},
	// GET /links/new: /newフォームからhtmxで取得するリンクカードプロンプトのフラグメント。
	{pattern: regexp.MustCompile(`^/links/new$`), methods: []string{http.MethodGet}},
	// POST /links: リンクカード作成。
	{pattern: regexp.MustCompile(`^/links$`), methods: []string{http.MethodPost}},
	// GET /settings: 設定メニューページ。"^/settings$" のアンカーにより、
	// サブページ (GET /settings/profile・/settings/user・/settings/email) はRailsに残す。
	{pattern: regexp.MustCompile(`^/settings$`), methods: []string{http.MethodGet}},
	// GET/POST /settings/export: エクスポート画面とエクスポートの開始。
	// Go版にしか存在しないルートのため、Go版が単独で持つ。
	{pattern: regexp.MustCompile(`^/settings/export$`), methods: []string{http.MethodGet, http.MethodPost}},
	// GET /settings/export/download: 生成したzipのダウンロード。
	{pattern: regexp.MustCompile(`^/settings/export/download$`), methods: []string{http.MethodGet}},
}

// NewReverseProxyMiddlewareは新しいReverseProxyMiddlewareを作成する。
// featureFlagRepoがnilの場合、フィーチャーフラグ判定はスキップされる。
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

	// プロキシのRewrite関数でヘッダー設定を行う。httputil.ReverseProxyはRewrite呼び出し前に
	// Forwarded / X-Forwarded-For / X-Forwarded-Host / X-Forwarded-ProtoをOut.Headerから削除するため、
	// 元の値を参照したい場合はpr.In.Headerから取得する必要がある。
	proxy.Rewrite = func(pr *httputil.ProxyRequest) {
		// URLをRails版のホストに書き換える。SetURLはOut.Host = "" をセットしてしまうため、
		// 続けてOut.Host = In.Hostを設定し、クライアントが送ってきたHostヘッダをそのままRails版に
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

		// Rails版が502を返したり接続が切れた場合はGo版の障害ではなく
		// Rails版の外部要因 (= Go版のSentryにIssueを作るほどではない) のためWarnに留める。
		// slog handlerはLevelError以上のみSentryに送るので、ここをWarnにすることで
		// Sentryへのノイズ送信を構造的に防止する。
		slog.WarnContext(ctx, "Rails版へのプロキシでエラーが発生",
			"error", err,
			"path", r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
		)

		// reverse_proxyはi18n.Middlewareより前に動くため、ロケールを検出してcontextに載せ替えてからhttperrorに委譲する
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

// MiddlewareはHTTPミドルウェアを返す。
func (m *ReverseProxyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 常にGo版で処理するパス (prefixホワイトリスト)。加えてprefix
		// 一致では粗すぎる完全一致 + メソッドのパターン (例: POST /postsはGo・
		// GET /posts/:idはRails) もここで処理する。
		if m.isGoHandledPath(r.URL.Path) || m.isGoHandledPattern(r) {
			next.ServeHTTP(w, r)
			return
		}

		// device_token Cookieが無ければ発行する。ログイン状態に依らず
		// ブラウザを識別してフィーチャーフラグ判定に使えるようにするため。
		// 静的アセットやヘルスチェック (フィーチャーフラグの出し分けが不要なパス) に
		// Set-Cookieを出さないよう、Go処理パスの判定より後で発行する。
		m.ensureDeviceToken(w, r)

		// 2. フィーチャーフラグで制御するパス。閲覧者に対してフラグが
		// 有効なときだけGo版で処理し、無効ならRailsへのプロキシに進む。
		if flagName := m.getFeatureFlagForRequest(r); flagName != "" {
			if m.isFeatureFlagEnabled(r, flagName) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// 3. それ以外はすべてRailsにプロキシする。
		m.proxy.ServeHTTP(w, r)
	})
}

// ensureDeviceTokenはリクエストにdevice_token Cookieが無ければ発行する。
func (m *ReverseProxyMiddleware) ensureDeviceToken(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(DeviceTokenCookieName); err == nil {
		return // 既にCookieが存在する
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
		MaxAge:   10 * 365 * 24 * 60 * 60, // 10年
		HttpOnly: true,
		Secure:   m.cfg.SessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// isGoHandledPathはGo版で処理するパスかどうかを判定
func (m *ReverseProxyMiddleware) isGoHandledPath(path string) bool {
	for _, p := range goHandledPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// isGoHandledPatternはリクエストがgoHandledPatternsのエントリに完全一致
// パスとメソッドで一致するか (= 常にGo版で処理する) を判定する。
func (m *ReverseProxyMiddleware) isGoHandledPattern(r *http.Request) bool {
	for _, gp := range goHandledPatterns {
		if matchesPattern(gp.pattern, gp.methods, r) {
			return true
		}
	}
	return false
}

// getFeatureFlagForRequestはリクエストのパスとメソッドに一致する
// フィーチャーフラグ名を返す。一致するパターンが無ければ空文字列を返す。
func (m *ReverseProxyMiddleware) getFeatureFlagForRequest(r *http.Request) model.FeatureFlagName {
	for _, fp := range featureFlaggedPatterns {
		if matchesPattern(fp.pattern, fp.methods, r) {
			return fp.flag
		}
	}
	return ""
}

// matchesPatternはリクエストのパスとメソッドが、指定したパス正規表現と
// メソッド集合に一致するかを判定する。メソッド集合が空ならすべてのメソッドに
// 一致する。goHandledPatterns / featureFlaggedPatternsに同じ「正規表現 +
// メソッド」のマッチを適用するisGoHandledPatternとgetFeatureFlagForRequestで
// 共有する。
func matchesPattern(pattern *regexp.Regexp, methods []string, r *http.Request) bool {
	if !pattern.MatchString(r.URL.Path) {
		return false
	}
	if len(methods) > 0 && !containsMethod(methods, r.Method) {
		return false
	}
	return true
}

// containsMethodはmethodがmethodsに含まれるかを判定する。
//
// HTMLフォームはGETとPOSTのみをサポートするため、PATCH/PUT/DELETEは
// POST + _methodパラメータとして送信される (Method Overrideパターン)。
// 本ミドルウェアはMethod Overrideミドルウェアより前に実行されるため、
// POSTリクエストもPATCH/PUT/DELETEパターンにマッチさせる必要がある。
func containsMethod(methods []string, method string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}

	// POSTリクエストはMethod Override経由でPATCH/PUT/DELETEに変換される可能性がある。
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

// isFeatureFlagEnabledはリクエストのCookieからフィーチャーフラグが
// 有効かどうかを判定する。エラー時または識別用Cookie不在時はfalseを返し、
// Rails版にフォールバックする。
func (m *ReverseProxyMiddleware) isFeatureFlagEnabled(r *http.Request, flagName model.FeatureFlagName) bool {
	if m.featureFlagRepo == nil {
		return false
	}

	// device_token Cookieの値を取得する。
	deviceToken := ""
	if cookie, err := r.Cookie(DeviceTokenCookieName); err == nil {
		deviceToken = cookie.Value
	}

	// Rails版と共有するセッショントークンCookieの値を取得する。
	sessionToken := ""
	if cookie, err := r.Cookie(session.CookieName); err == nil {
		sessionToken = cookie.Value
	}

	// どちらのCookieも存在しない場合はRails版にフォールバックする。
	if deviceToken == "" && sessionToken == "" {
		return false
	}

	// device_tokenとセッション経由のactorを1クエリで判定する。
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
