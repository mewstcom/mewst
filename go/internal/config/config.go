// Package configはアプリケーション設定の管理機能を提供します
package config

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Configはアプリケーションの設定を保持する構造体です
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

	// エクスポート用オブジェクトストレージ (S3互換 - Cloudflare R2)。
	// バケット・エンドポイント・アクセスキー・シークレットは4項目セットで必須とし、
	// リージョンは任意とする。
	S3BucketName      string
	S3Endpoint        string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3Region          string
}

// S3Readinessはエクスポート機能で使うS3互換オブジェクトストレージ
// (Cloudflare R2) の設定状態を表す。
type S3Readiness string

const (
	// S3ReadinessDisabledはMEWST_S3_* がすべて未設定の状態。server / workerは
	// エクスポート機能を無効化したまま起動できる。この状態で起動できるのは非本番の
	// 環境だけで、ローカル実行やテストがバケットの資格情報を持たなくて済むようにする。
	// 本番ではエクスポートがログイン中の全ユーザーに公開された機能のため、Loadが
	// この状態を拒否する。
	S3ReadinessDisabled S3Readiness = "disabled"

	// S3ReadinessReadyは必須のMEWST_S3_* がすべて設定され、エクスポート用
	// ストレージを使用できる状態。
	S3ReadinessReady S3Readiness = "ready"

	// S3ReadinessInvalidはMEWST_S3_* が一部だけ設定された構成ミスの状態。
	// Loadがエラーを返すためプロセスは起動しない。
	S3ReadinessInvalid S3Readiness = "invalid"
)

// Loadは環境変数から設定を読み込みます
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
		return nil, fmt.Errorf("必須の環境変数DATABASE_URLが設定されていません")
	}

	cfg.Port = os.Getenv("MEWST_PORT")
	if cfg.Port == "" {
		return nil, fmt.Errorf("必須の環境変数MEWST_PORTが設定されていません")
	}

	cfg.Domain = os.Getenv("MEWST_DOMAIN")
	if cfg.Domain == "" {
		return nil, fmt.Errorf("必須の環境変数MEWST_DOMAINが設定されていません")
	}

	cfg.CookieDomain = os.Getenv("MEWST_COOKIE_DOMAIN")
	if cfg.CookieDomain == "" {
		return nil, fmt.Errorf("必須の環境変数MEWST_COOKIE_DOMAINが設定されていません")
	}

	sessionSecureStr := os.Getenv("MEWST_SESSION_SECURE")
	if sessionSecureStr == "" {
		return nil, fmt.Errorf("必須の環境変数MEWST_SESSION_SECUREが設定されていません")
	}
	cfg.SessionSecure = sessionSecureStr == "true"

	sessionHTTPOnlyStr := os.Getenv("MEWST_SESSION_HTTPONLY")
	if sessionHTTPOnlyStr == "" {
		return nil, fmt.Errorf("必須の環境変数MEWST_SESSION_HTTPONLYが設定されていません")
	}
	cfg.SessionHTTPOnly = sessionHTTPOnlyStr == "true"

	// Rate Limiting設定 (オプショナル - 開発環境でRate Limitingを無効化)
	cfg.DisableRateLimit = os.Getenv("MEWST_DISABLE_RATE_LIMIT") == "true"

	// Rails版アプリのURL (オプショナル - リバースプロキシ機能で使用)
	cfg.RailsAppURL = os.Getenv("MEWST_RAILS_APP_URL")

	// Cloudflare Turnstile (オプショナル - ログイン・サインアップフォームで使用)。
	// テスト環境では空文字列でも動作する (モック設定として使用)。
	cfg.TurnstileSiteKey = os.Getenv("MEWST_TURNSTILE_SITE_KEY")
	cfg.TurnstileSecretKey = os.Getenv("MEWST_TURNSTILE_SECRET_KEY")

	// MEWST_TURNSTILE_DISABLE=trueは2つのキーを空に落とすことで非本番のTurnstileを
	// 1フラグで無効化する。空キーは既存経路 (Verifyは常に成功・ウィジェット非描画) にそのまま
	// 乗るため、turnstile.Verify / turnstile.templに新たな分岐は足さない。本番ではfail-closed
	// とする。TurnstileはBot対策のため、無効化フラグが誤って本番に漏れても黙って無効化されては
	// ならない。本番ではフラグを無視してキーを維持し、警告ログを出す。
	if os.Getenv("MEWST_TURNSTILE_DISABLE") == "true" {
		if cfg.IsProduction() {
			slog.Warn("MEWST_TURNSTILE_DISABLEは本番環境では無視されます (Turnstileキーは変更しません)")
		} else {
			cfg.TurnstileSiteKey = ""
			cfg.TurnstileSecretKey = ""
		}
	}

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
	// DSNが空のときはSentryを完全に無効化する
	cfg.SentryDSN = os.Getenv("MEWST_SENTRY_DSN")
	cfg.SentryEnvironment = os.Getenv("MEWST_SENTRY_ENVIRONMENT")
	if cfg.SentryEnvironment == "" {
		cfg.SentryEnvironment = env
	}
	cfg.SentryTracesSampleRate = parseSentryTracesSampleRate(os.Getenv("MEWST_SENTRY_TRACES_SAMPLE_RATE"))
	cfg.SentryDebug = os.Getenv("MEWST_SENTRY_DEBUG") == "true"

	// エクスポート用オブジェクトストレージ (S3互換 - Cloudflare R2)。
	// MEWST_S3_* の部分設定は構成ミスであり、本番では未設定も同じく構成ミスとなる。
	// どちらも後からエクスポートの故障として表面化させず、ここで起動を失敗させる。
	cfg.S3BucketName = os.Getenv("MEWST_S3_BUCKET_NAME")
	cfg.S3Endpoint = os.Getenv("MEWST_S3_ENDPOINT")
	cfg.S3AccessKeyID = os.Getenv("MEWST_S3_ACCESS_KEY_ID")
	cfg.S3SecretAccessKey = os.Getenv("MEWST_S3_SECRET_ACCESS_KEY")
	cfg.S3Region = os.Getenv("MEWST_S3_REGION")

	switch cfg.S3Readiness() {
	case S3ReadinessInvalid:
		return nil, fmt.Errorf("MEWST_S3_* 環境変数が一部だけ設定されています (未設定の必須項目: %s)。必須4項目を設定するか、MEWST_S3_REGIONを含む全項目を未設定にしてください", strings.Join(missingS3EnvVars(cfg), ", "))
	case S3ReadinessDisabled:
		if cfg.IsProduction() {
			return nil, fmt.Errorf("本番環境ではMEWST_S3_* 環境変数が必須です (未設定の必須項目: %s)。エクスポートは全ユーザーに公開された機能のため、ストレージ未設定のまま起動させない", strings.Join(missingS3EnvVars(cfg), ", "))
		}
	case S3ReadinessReady:
		if cfg.S3Region == "" {
			cfg.S3Region = "auto"
		}
	}

	return cfg, nil
}

// S3Readinessはエクスポート用オブジェクトストレージの設定状態を返す。
// バケット・エンドポイント・アクセスキー・シークレットは4項目セットで必須。
// リージョンは任意 (既定 "auto") だが、リージョンだけが設定された状態も
// 部分設定の構成ミス (invalid) として扱う。
func (c *Config) S3Readiness() S3Readiness {
	required := c.requiredS3Vars()
	setCount := 0
	for _, v := range required {
		if v.value != "" {
			setCount++
		}
	}

	switch {
	case setCount == len(required):
		return S3ReadinessReady
	case setCount == 0 && c.S3Region == "":
		return S3ReadinessDisabled
	default:
		return S3ReadinessInvalid
	}
}

// requiredS3Varsは必須のMEWST_S3_* の変数名と現在値の組を返す。
// readiness判定と部分設定エラーメッセージの両方がこの一覧を参照する。
func (c *Config) requiredS3Vars() []struct{ name, value string } {
	return []struct{ name, value string }{
		{"MEWST_S3_BUCKET_NAME", c.S3BucketName},
		{"MEWST_S3_ENDPOINT", c.S3Endpoint},
		{"MEWST_S3_ACCESS_KEY_ID", c.S3AccessKeyID},
		{"MEWST_S3_SECRET_ACCESS_KEY", c.S3SecretAccessKey},
	}
}

// missingS3EnvVarsは必須のMEWST_S3_* のうち未設定の変数名を返す。
// 部分設定エラーのメッセージに使う。
func missingS3EnvVars(c *Config) []string {
	vars := c.requiredS3Vars()

	missing := make([]string, 0, len(vars))
	for _, v := range vars {
		if v.value == "" {
			missing = append(missing, v.name)
		}
	}
	return missing
}

// DatabaseDSNはPostgreSQL接続文字列を返します
func (c *Config) DatabaseDSN() string {
	return c.DatabaseURL
}

// IsDevは開発環境かどうかを返します
func (c *Config) IsDev() bool {
	return c.Env == "dev"
}

// IsTestはテスト環境かどうかを返します
func (c *Config) IsTest() bool {
	return c.Env == "test"
}

// IsProductionは本番環境かどうかを返します
func (c *Config) IsProduction() bool {
	return c.Env == "prod"
}

// AppURLはアプリケーションのベースURLを返します
func (c *Config) AppURL() string {
	return "https://" + c.Domain
}

// 実行中ビルドのGitコミットハッシュ (短縮版) を返す。Sentryのreleaseと、
// CDNキャッシュ対策用のCSS/JSクエリパラメータに使う。
//
// GIT_REVを最優先する。Dokkuのデプロイ先コンテナには .gitディレクトリが無いため
// `git rev-parse` は失敗し、そのままだと "dev" にフォールバックしてしまう。Dokkuは
// 代わりにデプロイ時のコミットハッシュをGIT_REV環境変数で渡す (プラットフォームが
// 提供する変数なのでMEWST_ プレフィックスは付けない)。ローカルのgitコマンドは
// 開発用のフォールバックで、最後の手段が "dev"。
func getGitCommitHash() string {
	// Dokkuはここに完全なデプロイSHAを渡すので、7文字に短縮して
	// ローカルの `git rev-parse --short` が返す短縮形におおよそ揃える。
	if rev := strings.TrimSpace(os.Getenv("GIT_REV")); rev != "" {
		const shortHashLen = 7
		if len(rev) > shortHashLen {
			return rev[:shortHashLen]
		}
		return rev
	}

	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		// Gitが利用できない場合は "dev" を返す (開発環境用のフォールバック)。
		return "dev"
	}
	return strings.TrimSpace(string(out))
}

// GetAssetVersionはアセットのバージョン文字列を返します
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

// parseAdminIPsはカンマ区切りのIP文字列をスライスに変換します
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

// parseSentryTracesSampleRateは文字列からSentryトレースサンプリングレートをパースします
// 空文字列、パース失敗、範囲外 (0.0未満 または1.0超) の場合はデフォルト値0.5を返します
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
