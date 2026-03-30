package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/mewstcom/mewst/go/internal/clientip"
	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/pages/errors"
)

// ReverseProxyMiddleware はRails版へのリバースプロキシミドルウェア
type ReverseProxyMiddleware struct {
	railsURL *url.URL
	proxy    *httputil.ReverseProxy
	cfg      *config.Config
}

// Go版で処理するパス（ホワイトリスト）
// これらのパスはRails版にプロキシせず、Go版のハンドラーで処理する
var goHandledPaths = []string{
	"/static",             // 静的ファイル（CSS、JS、画像など）
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

// NewReverseProxyMiddleware は新しいReverseProxyMiddlewareを作成
func NewReverseProxyMiddleware(railsURL string, cfg *config.Config) (*ReverseProxyMiddleware, error) {
	parsedURL, err := url.Parse(railsURL)
	if err != nil {
		return nil, err
	}

	// httputil.ReverseProxyを作成
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	// カスタムのHTTP Transportを設定（タイムアウトと接続プーリング）
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

	// プロキシのディレクターをカスタマイズ（ヘッダー設定）
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// 既存のX-Forwarded-ForとX-Real-IPを保存
		originalXForwardedFor := req.Header.Get("X-Forwarded-For")
		originalXRealIP := req.Header.Get("X-Real-IP")

		// クライアントIPアドレスを取得
		clientIP := clientip.GetClientIP(req)

		// originalDirectorを呼び出す
		originalDirector(req)

		// X-Forwarded-Forヘッダーの設定
		// 注: httputil.ReverseProxyのServeHTTPメソッドは、Directorを呼び出した後に
		// X-Forwarded-Forヘッダーが存在する場合、RemoteAddrを追加してしまう。
		// これを防ぐために、ヘッダーマップから完全に削除してから再設定する。
		delete(req.Header, "X-Forwarded-For")
		if originalXForwardedFor != "" {
			// 既存の値を維持（Cloudflareなどが設定した値を保持）
			req.Header.Set("X-Forwarded-For", originalXForwardedFor)
		} else {
			// 既存の値がない場合、clientIPを設定
			req.Header.Set("X-Forwarded-For", clientIP)
		}

		// X-Real-IPヘッダーの設定（既存の値がない場合のみ）
		if originalXRealIP != "" {
			req.Header.Set("X-Real-IP", originalXRealIP)
		} else {
			req.Header.Set("X-Real-IP", clientIP)
		}

		// X-Forwarded-Protoの設定
		req.Header.Set("X-Forwarded-Proto", "https")

		// X-Forwarded-Hostの設定
		req.Header.Set("X-Forwarded-Host", cfg.Domain)

		// ログ出力（開発者向け）
		slog.Info("リバースプロキシでRails版にリクエストを転送",
			"path", req.URL.Path,
			"method", req.Method,
			"target", parsedURL.String()+req.URL.Path,
			"client_ip", clientIP,
		)
	}

	// レスポンス処理後のログ出力（成功時）
	proxy.ModifyResponse = func(resp *http.Response) error {
		// プロキシが成功した場合のレスポンスログを出力（開発者向け）
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

		// 詳細なエラーログを出力（開発者向け）
		slog.ErrorContext(ctx, "Rails版へのプロキシでエラーが発生",
			"error", err,
			"path", r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
		)

		// 502エラーレスポンスを返す
		locale := i18n.DetectLanguage(r)
		ctx = templates.WithLocale(ctx, locale)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)

		if err := errors.BadGateway().Render(ctx, w); err != nil {
			slog.ErrorContext(ctx, "502ページのレンダリングに失敗", "error", err)
		}
	}

	return &ReverseProxyMiddleware{
		railsURL: parsedURL,
		proxy:    proxy,
		cfg:      cfg,
	}, nil
}

// Middleware はHTTPミドルウェアを返す
func (m *ReverseProxyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Go版で処理するパスかどうかをチェック
		if m.isGoHandledPath(r.URL.Path) {
			// Go版で処理する
			next.ServeHTTP(w, r)
			return
		}

		// Rails版にプロキシ
		m.proxy.ServeHTTP(w, r)
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
