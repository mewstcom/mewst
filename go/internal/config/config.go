// Package config はアプリケーション設定の管理機能を提供します
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

	// Object storage for exports (S3-compatible - Cloudflare R2). The bucket,
	// endpoint, access key and secret must be set together; the region is optional.
	//
	// [Ja] エクスポート用オブジェクトストレージ (S3 互換 - Cloudflare R2)。
	// バケット・エンドポイント・アクセスキー・シークレットは 4 項目セットで必須とし、
	// リージョンは任意とする。
	S3BucketName      string
	S3Endpoint        string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3Region          string
}

// S3Readiness represents the configuration state of the S3-compatible object
// storage (Cloudflare R2) used by the export feature.
//
// [Ja] S3Readiness はエクスポート機能で使う S3 互換オブジェクトストレージ
// (Cloudflare R2) の設定状態を表す。
type S3Readiness string

const (
	// S3ReadinessDisabled means every MEWST_S3_* variable is unset: the server
	// and worker boot with the export feature disabled. A flag-off deploy must
	// not require the R2 configuration.
	//
	// [Ja] S3ReadinessDisabled は MEWST_S3_* がすべて未設定の状態。server / worker は
	// エクスポート機能を無効化したまま起動できる。フラグ OFF のデプロイに R2 設定を
	// 必須化しない。
	S3ReadinessDisabled S3Readiness = "disabled"

	// S3ReadinessReady means every required MEWST_S3_* variable is set and the
	// export storage can be used.
	//
	// [Ja] S3ReadinessReady は必須の MEWST_S3_* がすべて設定され、エクスポート用
	// ストレージを使用できる状態。
	S3ReadinessReady S3Readiness = "ready"

	// S3ReadinessInvalid means only part of MEWST_S3_* is set, which is a
	// configuration mistake. Load fails so the process does not boot.
	//
	// [Ja] S3ReadinessInvalid は MEWST_S3_* が一部だけ設定された構成ミスの状態。
	// Load がエラーを返すためプロセスは起動しない。
	S3ReadinessInvalid S3Readiness = "invalid"
)

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

	// Cloudflare Turnstile (optional - used by the sign-in and sign-up forms).
	// An empty key works in the test environment too (used as a mock config).
	//
	// [Ja] Cloudflare Turnstile (オプショナル - ログイン・サインアップフォームで使用)。
	// テスト環境では空文字列でも動作する (モック設定として使用)。
	cfg.TurnstileSiteKey = os.Getenv("MEWST_TURNSTILE_SITE_KEY")
	cfg.TurnstileSecretKey = os.Getenv("MEWST_TURNSTILE_SECRET_KEY")

	// MEWST_TURNSTILE_DISABLE=true disables Turnstile outside production with a single
	// flag by blanking both keys, which routes through the existing "empty key"
	// path (Verify always succeeds and the widget is not rendered), so no new
	// branch is added to turnstile.Verify or turnstile.templ. It is fail-closed in
	// production: Turnstile is a bot countermeasure, so a stray disable flag must
	// never silently turn it off. In production the flag is ignored, the keys are
	// kept, and a warning is logged.
	//
	// [Ja] MEWST_TURNSTILE_DISABLE=true は 2 つのキーを空に落とすことで非本番の Turnstile を
	// 1 フラグで無効化する。空キーは既存経路 (Verify は常に成功・ウィジェット非描画) にそのまま
	// 乗るため、turnstile.Verify / turnstile.templ に新たな分岐は足さない。本番では fail-closed
	// とする。Turnstile は Bot 対策のため、無効化フラグが誤って本番に漏れても黙って無効化されては
	// ならない。本番ではフラグを無視してキーを維持し、警告ログを出す。
	if os.Getenv("MEWST_TURNSTILE_DISABLE") == "true" {
		if cfg.IsProduction() {
			slog.Warn("MEWST_TURNSTILE_DISABLE は本番環境では無視されます (Turnstile キーは変更しません)")
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
	// DSN が空のときは Sentry を完全に無効化する
	cfg.SentryDSN = os.Getenv("MEWST_SENTRY_DSN")
	cfg.SentryEnvironment = os.Getenv("MEWST_SENTRY_ENVIRONMENT")
	if cfg.SentryEnvironment == "" {
		cfg.SentryEnvironment = env
	}
	cfg.SentryTracesSampleRate = parseSentryTracesSampleRate(os.Getenv("MEWST_SENTRY_TRACES_SAMPLE_RATE"))
	cfg.SentryDebug = os.Getenv("MEWST_SENTRY_DEBUG") == "true"

	// Object storage for exports (S3-compatible - Cloudflare R2). A partial
	// MEWST_S3_* set is a configuration mistake: fail the boot here instead of
	// letting it surface later as a broken export.
	//
	// [Ja] エクスポート用オブジェクトストレージ (S3 互換 - Cloudflare R2)。
	// MEWST_S3_* の部分設定は構成ミスのため、後からエクスポートの故障として
	// 表面化させず、ここで起動を失敗させる。
	cfg.S3BucketName = os.Getenv("MEWST_S3_BUCKET_NAME")
	cfg.S3Endpoint = os.Getenv("MEWST_S3_ENDPOINT")
	cfg.S3AccessKeyID = os.Getenv("MEWST_S3_ACCESS_KEY_ID")
	cfg.S3SecretAccessKey = os.Getenv("MEWST_S3_SECRET_ACCESS_KEY")
	cfg.S3Region = os.Getenv("MEWST_S3_REGION")

	switch cfg.S3Readiness() {
	case S3ReadinessInvalid:
		return nil, fmt.Errorf("MEWST_S3_* 環境変数が一部だけ設定されています (未設定の必須項目: %s)。必須 4 項目を設定するか、MEWST_S3_REGION を含む全項目を未設定にしてください", strings.Join(missingS3EnvVars(cfg), ", "))
	case S3ReadinessReady:
		if cfg.S3Region == "" {
			cfg.S3Region = "auto"
		}
	}

	return cfg, nil
}

// S3Readiness reports the export object storage configuration state. The
// bucket, endpoint, access key and secret are required as a set. The region is
// optional (it defaults to "auto"), but a region set on its own still counts as
// a partial, invalid configuration.
//
// [Ja] S3Readiness はエクスポート用オブジェクトストレージの設定状態を返す。
// バケット・エンドポイント・アクセスキー・シークレットは 4 項目セットで必須。
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

// requiredS3Vars returns the required MEWST_S3_* variable names and their
// current values as the single source for both the readiness check and the
// partial-configuration error message.
//
// [Ja] requiredS3Vars は必須の MEWST_S3_* の変数名と現在値の組を返す。
// readiness 判定と部分設定エラーメッセージの両方がこの一覧を参照する。
func (c *Config) requiredS3Vars() []struct{ name, value string } {
	return []struct{ name, value string }{
		{"MEWST_S3_BUCKET_NAME", c.S3BucketName},
		{"MEWST_S3_ENDPOINT", c.S3Endpoint},
		{"MEWST_S3_ACCESS_KEY_ID", c.S3AccessKeyID},
		{"MEWST_S3_SECRET_ACCESS_KEY", c.S3SecretAccessKey},
	}
}

// missingS3EnvVars returns the names of the unset variables among the required
// MEWST_S3_* set, for the partial-configuration error message.
//
// [Ja] missingS3EnvVars は必須の MEWST_S3_* のうち未設定の変数名を返す。
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

// getGitCommitHash returns the short Git commit hash of the running build. It
// is used as the Sentry release and as the CSS/JS query parameter for CDN cache
// busting.
//
// GIT_REV takes precedence: on Dokku the deployed container has no .git
// directory, so `git rev-parse` fails there and the value would fall back to
// "dev". Dokku instead exposes the deploy commit hash via the GIT_REV
// environment variable, which is provided by the platform (so it carries no
// MEWST_ prefix). The local git command is the development fallback, and "dev"
// is the last resort.
//
// [Ja] 実行中ビルドの Git コミットハッシュ (短縮版) を返す。Sentry の release と、
// CDN キャッシュ対策用の CSS/JS クエリパラメータに使う。
//
// GIT_REV を最優先する。Dokku のデプロイ先コンテナには .git ディレクトリが無いため
// `git rev-parse` は失敗し、そのままだと "dev" にフォールバックしてしまう。Dokku は
// 代わりにデプロイ時のコミットハッシュを GIT_REV 環境変数で渡す (プラットフォームが
// 提供する変数なので MEWST_ プレフィックスは付けない)。ローカルの git コマンドは
// 開発用のフォールバックで、最後の手段が "dev"。
func getGitCommitHash() string {
	// Dokku provides the full deploy SHA here; shorten it to 7 characters to
	// roughly match the abbreviated form the local `git rev-parse --short` path
	// produces.
	//
	// [Ja] Dokku はここに完全なデプロイ SHA を渡すので、7 文字に短縮して
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
		// Fall back to "dev" when git is unavailable (development environment).
		//
		// [Ja] Git が利用できない場合は "dev" を返す (開発環境用のフォールバック)。
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
