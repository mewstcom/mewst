// Package config はアプリケーション設定の管理機能を提供します
package config

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Config はアプリケーションの設定を保持する構造体です
type Config struct {
	// 環境
	Env string

	// データベース
	DatabaseURL string

	// サーバー
	Port   string
	Domain string

	// Cookie設定
	CookieDomain string

	// セッション
	SessionSecure   bool
	SessionHTTPOnly bool

	// Rate Limiting設定
	DisableRateLimit bool

	// Rails版アプリのURL (リバースプロキシ用)
	RailsAppURL string

	// Cloudflare Turnstile (Bot対策)
	TurnstileSiteKey   string
	TurnstileSecretKey string

	// メンテナンスモード
	MaintenanceMode bool
	AdminIPs        []string

	// アセットバージョン (CDNキャッシュ対策用)
	AssetVersion string

	// メール送信 (Resend API)
	ResendAPIKey  string
	EmailFrom     string
	EmailFromName string

	// Sentry (エラー追跡)
	SentryDSN              string
	SentryEnvironment      string
	SentryTracesSampleRate float64
	SentryDebug            bool
}

// Load は環境変数から設定を読み込みます
func Load() (*Config, error) {
	// APP_ENVの値を取得 (デフォルト: dev)
	// dev: 開発環境、test: テスト環境、prod: 本番環境
	//
	// すべての環境でGoプロセス起動時には既に環境変数がセット済みです：
	// - ローカル開発/テスト: op run --env-file=".env" が処理済み
	// - CI環境: GitHub Actionsが設定済み
	// - 本番環境: Dokkuが設定済み
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	cfg := &Config{
		Env: env,
	}

	// 必須の環境変数をチェック
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("必須の環境変数 DATABASE_URL が設定されていません")
	}

	cfg.Port = os.Getenv("MEWST_PORT")
	if cfg.Port == "" {
		return nil, fmt.Errorf("必須の環境変数 MEWST_PORT が設定されていません")
	}

	cfg.Domain = os.Getenv("MEWST_DOMAIN")
	if cfg.Domain == "" {
		return nil, fmt.Errorf("必須の環境変数 MEWST_DOMAIN が設定されていません")
	}

	cfg.CookieDomain = os.Getenv("MEWST_COOKIE_DOMAIN")
	if cfg.CookieDomain == "" {
		return nil, fmt.Errorf("必須の環境変数 MEWST_COOKIE_DOMAIN が設定されていません")
	}

	sessionSecureStr := os.Getenv("MEWST_SESSION_SECURE")
	if sessionSecureStr == "" {
		return nil, fmt.Errorf("必須の環境変数 MEWST_SESSION_SECURE が設定されていません")
	}
	cfg.SessionSecure = sessionSecureStr == "true"

	sessionHTTPOnlyStr := os.Getenv("MEWST_SESSION_HTTPONLY")
	if sessionHTTPOnlyStr == "" {
		return nil, fmt.Errorf("必須の環境変数 MEWST_SESSION_HTTPONLY が設定されていません")
	}
	cfg.SessionHTTPOnly = sessionHTTPOnlyStr == "true"

	// Rate Limiting設定 (オプショナル - 開発環境でRate Limitingを無効化)
	cfg.DisableRateLimit = os.Getenv("MEWST_DISABLE_RATE_LIMIT") == "true"

	// Rails版アプリのURL (オプショナル - リバースプロキシ機能で使用)
	cfg.RailsAppURL = os.Getenv("MEWST_RAILS_APP_URL")

	// Cloudflare Turnstile (オプショナル - ログイン・サインアップフォームで使用)
	// テスト環境では空文字列でも動作する (モック設定として使用)
	cfg.TurnstileSiteKey = os.Getenv("MEWST_TURNSTILE_SITE_KEY")
	cfg.TurnstileSecretKey = os.Getenv("MEWST_TURNSTILE_SECRET_KEY")

	// メンテナンスモード (オプショナル - "on"のときメンテナンスモードを有効化)
	cfg.MaintenanceMode = os.Getenv("MEWST_MAINTENANCE_MODE") == "on"

	// 管理者IP (オプショナル - カンマ区切りで複数指定可能)
	adminIPStr := os.Getenv("MEWST_ADMIN_IP")
	if adminIPStr != "" {
		cfg.AdminIPs = parseAdminIPs(adminIPStr)
	}

	// アセットバージョン (Gitコミットハッシュ) を設定
	cfg.AssetVersion = getGitCommitHash()

	// Resend API (オプショナル - メール送信で使用)
	// テスト環境では空文字列でも動作する (モックSenderを使用)
	cfg.ResendAPIKey = os.Getenv("MEWST_RESEND_API_KEY")
	cfg.EmailFrom = os.Getenv("MEWST_EMAIL_FROM")
	cfg.EmailFromName = os.Getenv("MEWST_EMAIL_FROM_NAME")

	// Sentry (オプショナル - エラー追跡サービス)
	// DSN が空のときは Sentry を完全に無効化する
	cfg.SentryDSN = os.Getenv("MEWST_SENTRY_DSN")
	cfg.SentryEnvironment = os.Getenv("MEWST_SENTRY_ENVIRONMENT")
	if cfg.SentryEnvironment == "" {
		cfg.SentryEnvironment = env
	}
	cfg.SentryTracesSampleRate = parseSentryTracesSampleRate(os.Getenv("MEWST_SENTRY_TRACES_SAMPLE_RATE"))
	cfg.SentryDebug = os.Getenv("MEWST_SENTRY_DEBUG") == "true"

	return cfg, nil
}

// DatabaseDSN は PostgreSQL 接続文字列を返します
func (c *Config) DatabaseDSN() string {
	return c.DatabaseURL
}

// IsDev は開発環境かどうかを返します
func (c *Config) IsDev() bool {
	return c.Env == "dev"
}

// IsTest はテスト環境かどうかを返します
func (c *Config) IsTest() bool {
	return c.Env == "test"
}

// IsProduction は本番環境かどうかを返します
func (c *Config) IsProduction() bool {
	return c.Env == "prod"
}

// AppURL はアプリケーションのベースURLを返します
func (c *Config) AppURL() string {
	return "https://" + c.Domain
}

// getGitCommitHash はGitのコミットハッシュ (短縮版) を取得します
// CDNキャッシュ対策として、CSS/JSファイルのクエリパラメータに使用します
func getGitCommitHash() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		// Gitが利用できない場合は "dev" を返す (開発環境用のフォールバック)
		return "dev"
	}
	return strings.TrimSpace(string(out))
}

// GetAssetVersion はアセットのバージョン文字列を返します
// 開発環境: 現在時刻のUnixタイムスタンプ (ミリ秒) を返す (キャッシュを無効化)
// 本番/テスト環境: Gitコミットハッシュを返す (起動時に設定された値)
func (c *Config) GetAssetVersion() string {
	if c.IsDev() {
		// 開発環境では毎回異なる値を返す (現在時刻のUnixタイムスタンプ、ミリ秒)
		return strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	// 本番/テスト環境では起動時に設定されたGitコミットハッシュを返す
	return c.AssetVersion
}

// parseAdminIPs はカンマ区切りのIP文字列をスライスに変換します
// 各IPアドレスの前後の空白は除去されます
func parseAdminIPs(s string) []string {
	parts := strings.Split(s, ",")
	ips := make([]string, 0, len(parts))
	for _, p := range parts {
		ip := strings.TrimSpace(p)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}

// parseSentryTracesSampleRate は文字列からSentryトレースサンプリングレートをパースします
// 空文字列、パース失敗、範囲外 (0.0未満 または 1.0超) の場合はデフォルト値 0.5 を返します
func parseSentryTracesSampleRate(s string) float64 {
	if s == "" {
		return 0.5
	}
	rate, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.5
	}
	if rate < 0.0 || rate > 1.0 {
		return 0.5
	}
	return rate
}
