package config

import (
	"os"
	"reflect"
	"testing"
)

// setupTestEnvは必須の環境変数を設定するヘルパー関数です
func setupTestEnv(t *testing.T) func() {
	t.Helper()

	// 既存の環境変数を保存
	savedEnvs := map[string]string{
		"APP_ENV":                         os.Getenv("APP_ENV"),
		"DATABASE_URL":                    os.Getenv("DATABASE_URL"),
		"MEWST_PORT":                      os.Getenv("MEWST_PORT"),
		"MEWST_DOMAIN":                    os.Getenv("MEWST_DOMAIN"),
		"MEWST_COOKIE_DOMAIN":             os.Getenv("MEWST_COOKIE_DOMAIN"),
		"MEWST_SESSION_SECURE":            os.Getenv("MEWST_SESSION_SECURE"),
		"MEWST_SESSION_HTTPONLY":          os.Getenv("MEWST_SESSION_HTTPONLY"),
		"MEWST_DISABLE_RATE_LIMIT":        os.Getenv("MEWST_DISABLE_RATE_LIMIT"),
		"MEWST_RAILS_APP_URL":             os.Getenv("MEWST_RAILS_APP_URL"),
		"MEWST_TURNSTILE_SITE_KEY":        os.Getenv("MEWST_TURNSTILE_SITE_KEY"),
		"MEWST_TURNSTILE_SECRET_KEY":      os.Getenv("MEWST_TURNSTILE_SECRET_KEY"),
		"MEWST_TURNSTILE_DISABLE":         os.Getenv("MEWST_TURNSTILE_DISABLE"),
		"MEWST_MAINTENANCE_MODE":          os.Getenv("MEWST_MAINTENANCE_MODE"),
		"MEWST_ADMIN_IP":                  os.Getenv("MEWST_ADMIN_IP"),
		"MEWST_SENTRY_DSN":                os.Getenv("MEWST_SENTRY_DSN"),
		"MEWST_SENTRY_ENVIRONMENT":        os.Getenv("MEWST_SENTRY_ENVIRONMENT"),
		"MEWST_SENTRY_TRACES_SAMPLE_RATE": os.Getenv("MEWST_SENTRY_TRACES_SAMPLE_RATE"),
		"MEWST_SENTRY_DEBUG":              os.Getenv("MEWST_SENTRY_DEBUG"),
		"MEWST_S3_BUCKET_NAME":            os.Getenv("MEWST_S3_BUCKET_NAME"),
		"MEWST_S3_ENDPOINT":               os.Getenv("MEWST_S3_ENDPOINT"),
		"MEWST_S3_ACCESS_KEY_ID":          os.Getenv("MEWST_S3_ACCESS_KEY_ID"),
		"MEWST_S3_SECRET_ACCESS_KEY":      os.Getenv("MEWST_S3_SECRET_ACCESS_KEY"),
		"MEWST_S3_REGION":                 os.Getenv("MEWST_S3_REGION"),
	}

	// 必須の環境変数を設定
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/mewst_test")
	_ = os.Setenv("MEWST_PORT", "3000")
	_ = os.Setenv("MEWST_DOMAIN", "test.mewst.com")
	_ = os.Setenv("MEWST_COOKIE_DOMAIN", ".test.mewst.com")
	_ = os.Setenv("MEWST_SESSION_SECURE", "false")
	_ = os.Setenv("MEWST_SESSION_HTTPONLY", "true")

	// Sentry関連は各テストが独立して状態を制御できるよう、デフォルトでは未設定にする
	_ = os.Unsetenv("MEWST_SENTRY_DSN")
	_ = os.Unsetenv("MEWST_SENTRY_ENVIRONMENT")
	_ = os.Unsetenv("MEWST_SENTRY_TRACES_SAMPLE_RATE")
	_ = os.Unsetenv("MEWST_SENTRY_DEBUG")

	// 各テストがMEWST_TURNSTILE_DISABLEを明示的に制御し、実 .envがconfigテストの
	// 挙動を変えないよう、デフォルトでは未設定にする。
	_ = os.Unsetenv("MEWST_TURNSTILE_DISABLE")

	// 各テストがエクスポート用ストレージ設定を明示的に制御し、実 .envが
	// configテストの挙動を変えないよう、MEWST_S3_* はデフォルトでは未設定にする。
	_ = os.Unsetenv("MEWST_S3_BUCKET_NAME")
	_ = os.Unsetenv("MEWST_S3_ENDPOINT")
	_ = os.Unsetenv("MEWST_S3_ACCESS_KEY_ID")
	_ = os.Unsetenv("MEWST_S3_SECRET_ACCESS_KEY")
	_ = os.Unsetenv("MEWST_S3_REGION")

	// クリーンアップ関数を返す
	return func() {
		for key, value := range savedEnvs {
			if value != "" {
				_ = os.Setenv(key, value)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}
}

// setExportStorageEnvはLoadが本番で必須とするMEWST_S3_* を設定する。
// 別のもの (Turnstile・Sentry) を見るために本番構成で起動するテストもこの要求を
// 満たす必要があるため、4項目を1回の呼び出しにまとめ、それらのテストの
// テーブルから必須項目を締め出す。
func setExportStorageEnv(t *testing.T) {
	t.Helper()

	_ = os.Setenv("MEWST_S3_BUCKET_NAME", "mewst-export-config-test")
	_ = os.Setenv("MEWST_S3_ENDPOINT", "https://example.r2.cloudflarestorage.com")
	_ = os.Setenv("MEWST_S3_ACCESS_KEY_ID", "config-test-access-key-id")
	_ = os.Setenv("MEWST_S3_SECRET_ACCESS_KEY", "config-test-secret-access-key")
}

// TestLoadは環境変数から設定を読み込むテスト
func TestLoad(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load()のエラー = %v", err)
	}

	// 基本的な設定が読み込まれていることを確認
	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL = 空文字列、非空を期待")
	}
	if cfg.Port == "" {
		t.Error("Port = 空文字列、非空を期待")
	}
	if cfg.Env != "test" {
		t.Errorf("Env = %v、期待値 = test", cfg.Env)
	}
	if cfg.Domain != "test.mewst.com" {
		t.Errorf("Domain = %v、期待値 = test.mewst.com", cfg.Domain)
	}
	if cfg.CookieDomain != ".test.mewst.com" {
		t.Errorf("CookieDomain = %v、期待値 = .test.mewst.com", cfg.CookieDomain)
	}
	if cfg.SessionSecure != false {
		t.Errorf("SessionSecure = %v、期待値 = false", cfg.SessionSecure)
	}
	if cfg.SessionHTTPOnly != true {
		t.Errorf("SessionHTTPOnly = %v、期待値 = true", cfg.SessionHTTPOnly)
	}
}

// TestLoad_MissingDatabaseURLはDATABASE_URLが未設定の場合のエラーをテスト
func TestLoad_MissingDatabaseURL(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("DATABASE_URL")

	_, err := Load()
	if err == nil {
		t.Error("DATABASE_URLが無いとき、Load()がエラーを返すことを期待したが、nilだった")
	}
}

// TestLoad_MissingPortはMEWST_PORTが未設定の場合のエラーをテスト
func TestLoad_MissingPort(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("MEWST_PORT")

	_, err := Load()
	if err == nil {
		t.Error("MEWST_PORTが無いとき、Load()がエラーを返すことを期待したが、nilだった")
	}
}

// TestLoad_MissingDomainはMEWST_DOMAINが未設定の場合のエラーをテスト
func TestLoad_MissingDomain(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("MEWST_DOMAIN")

	_, err := Load()
	if err == nil {
		t.Error("MEWST_DOMAINが無いとき、Load()がエラーを返すことを期待したが、nilだった")
	}
}

// TestLoad_MissingCookieDomainはMEWST_COOKIE_DOMAINが未設定の場合のエラーをテスト
func TestLoad_MissingCookieDomain(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("MEWST_COOKIE_DOMAIN")

	_, err := Load()
	if err == nil {
		t.Error("MEWST_COOKIE_DOMAINが無いとき、Load()がエラーを返すことを期待したが、nilだった")
	}
}

// TestLoad_MissingSessionSecureはMEWST_SESSION_SECUREが未設定の場合のエラーをテスト
func TestLoad_MissingSessionSecure(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("MEWST_SESSION_SECURE")

	_, err := Load()
	if err == nil {
		t.Error("MEWST_SESSION_SECUREが無いとき、Load()がエラーを返すことを期待したが、nilだった")
	}
}

// TestLoad_MissingSessionHTTPOnlyはMEWST_SESSION_HTTPONLYが未設定の場合のエラーをテスト
func TestLoad_MissingSessionHTTPOnly(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("MEWST_SESSION_HTTPONLY")

	_, err := Load()
	if err == nil {
		t.Error("MEWST_SESSION_HTTPONLYが無いとき、Load()がエラーを返すことを期待したが、nilだった")
	}
}

// TestLoad_DefaultEnvはAPP_ENVが未設定の場合のデフォルト値をテスト
func TestLoad_DefaultEnv(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("APP_ENV")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load()のエラー = %v", err)
	}

	if cfg.Env != "dev" {
		t.Errorf("Env = %v、期待値 = dev (既定値)", cfg.Env)
	}
}

// TestDatabaseDSNはDatabaseDSNメソッドをテスト
func TestDatabaseDSN(t *testing.T) {
	cfg := &Config{
		DatabaseURL: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
	}

	dsn := cfg.DatabaseDSN()
	expected := "postgres://user:pass@localhost:5432/testdb?sslmode=disable"

	if dsn != expected {
		t.Errorf("DatabaseDSN() = %v、期待値 = %v", dsn, expected)
	}
}

// TestIsDevはIsDevメソッドをテスト
func TestIsDev(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"dev", true},
		{"test", false},
		{"prod", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			if got := cfg.IsDev(); got != tt.want {
				t.Errorf("IsDev() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

// TestIsTestはIsTestメソッドをテスト
func TestIsTest(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"dev", false},
		{"test", true},
		{"prod", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			if got := cfg.IsTest(); got != tt.want {
				t.Errorf("IsTest() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

// TestIsProductionはIsProductionメソッドをテスト
func TestIsProduction(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"dev", false},
		{"test", false},
		{"prod", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			if got := cfg.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

// TestAppURLはAppURLメソッドをテスト
func TestAppURL(t *testing.T) {
	tests := []struct {
		env    string
		domain string
		want   string
	}{
		{"dev", "localhost", "https://localhost"},
		{"test", "test.mewst.com", "https://test.mewst.com"},
		{"prod", "mewst.com", "https://mewst.com"},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env, Domain: tt.domain}
			if got := cfg.AppURL(); got != tt.want {
				t.Errorf("AppURL() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

// TestLoad_SessionSecureはMEWST_SESSION_SECUREのbool変換をテスト
func TestLoad_SessionSecure(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"other", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			_ = os.Setenv("MEWST_SESSION_SECURE", tt.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}

			if cfg.SessionSecure != tt.want {
				t.Errorf("SessionSecure = %v、期待値 = %v", cfg.SessionSecure, tt.want)
			}
		})
	}
}

// TestLoad_SessionHTTPOnlyはMEWST_SESSION_HTTPONLYのbool変換をテスト
func TestLoad_SessionHTTPOnly(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"other", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			_ = os.Setenv("MEWST_SESSION_HTTPONLY", tt.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}

			if cfg.SessionHTTPOnly != tt.want {
				t.Errorf("SessionHTTPOnly = %v、期待値 = %v", cfg.SessionHTTPOnly, tt.want)
			}
		})
	}
}

// TestLoad_TurnstileConfigはTurnstile環境変数の読み込みをテスト
func TestLoad_TurnstileConfig(t *testing.T) {
	tests := []struct {
		name          string
		siteKey       string
		secretKey     string
		wantSiteKey   string
		wantSecretKey string
	}{
		{
			name:          "両方設定",
			siteKey:       "1x00000000000000000000AA",
			secretKey:     "1x0000000000000000000000000000000AA",
			wantSiteKey:   "1x00000000000000000000AA",
			wantSecretKey: "1x0000000000000000000000000000000AA",
		},
		{
			name:          "未設定",
			siteKey:       "",
			secretKey:     "",
			wantSiteKey:   "",
			wantSecretKey: "",
		},
		{
			name:          "Site Keyのみ設定",
			siteKey:       "1x00000000000000000000AA",
			secretKey:     "",
			wantSiteKey:   "1x00000000000000000000AA",
			wantSecretKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			if tt.siteKey != "" {
				_ = os.Setenv("MEWST_TURNSTILE_SITE_KEY", tt.siteKey)
			} else {
				_ = os.Unsetenv("MEWST_TURNSTILE_SITE_KEY")
			}
			if tt.secretKey != "" {
				_ = os.Setenv("MEWST_TURNSTILE_SECRET_KEY", tt.secretKey)
			} else {
				_ = os.Unsetenv("MEWST_TURNSTILE_SECRET_KEY")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}

			if cfg.TurnstileSiteKey != tt.wantSiteKey {
				t.Errorf("TurnstileSiteKey = %q、期待値 = %q", cfg.TurnstileSiteKey, tt.wantSiteKey)
			}
			if cfg.TurnstileSecretKey != tt.wantSecretKey {
				t.Errorf("TurnstileSecretKey = %q、期待値 = %q", cfg.TurnstileSecretKey, tt.wantSecretKey)
			}
		})
	}
}

// MEWST_TURNSTILE_DISABLEによる非本番でのワンフラグ無効化と、
// 本番でのfail-closedの挙動を検証する。
func TestLoad_TurnstileDisable(t *testing.T) {
	const (
		siteKey   = "1x00000000000000000000AA"
		secretKey = "1x0000000000000000000000000000000AA"
	)

	tests := []struct {
		name          string
		appEnv        string
		disable       string
		wantSiteKey   string
		wantSecretKey string
	}{
		{
			name:          "非prod + true: キーを空に落として無効化",
			appEnv:        "dev",
			disable:       "true",
			wantSiteKey:   "",
			wantSecretKey: "",
		},
		{
			name:          "prod + true: 無視してキーを維持 (fail-closed)",
			appEnv:        "prod",
			disable:       "true",
			wantSiteKey:   siteKey,
			wantSecretKey: secretKey,
		},
		{
			name:          "未設定: 既定どおりキーを維持",
			appEnv:        "dev",
			disable:       "",
			wantSiteKey:   siteKey,
			wantSecretKey: secretKey,
		},
		{
			name:          "true以外の値は無効化しない",
			appEnv:        "dev",
			disable:       "yes",
			wantSiteKey:   siteKey,
			wantSecretKey: secretKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			_ = os.Setenv("APP_ENV", tt.appEnv)
			if tt.appEnv == "prod" {
				setExportStorageEnv(t)
			}
			_ = os.Setenv("MEWST_TURNSTILE_SITE_KEY", siteKey)
			_ = os.Setenv("MEWST_TURNSTILE_SECRET_KEY", secretKey)
			if tt.disable != "" {
				_ = os.Setenv("MEWST_TURNSTILE_DISABLE", tt.disable)
			} else {
				_ = os.Unsetenv("MEWST_TURNSTILE_DISABLE")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}

			if cfg.TurnstileSiteKey != tt.wantSiteKey {
				t.Errorf("TurnstileSiteKey = %q、期待値 = %q", cfg.TurnstileSiteKey, tt.wantSiteKey)
			}
			if cfg.TurnstileSecretKey != tt.wantSecretKey {
				t.Errorf("TurnstileSecretKey = %q、期待値 = %q", cfg.TurnstileSecretKey, tt.wantSecretKey)
			}
		})
	}
}

// TestLoad_MaintenanceModeは メンテナンスモード設定のテスト
func TestLoad_MaintenanceMode(t *testing.T) {
	tests := []struct {
		name                string
		maintenanceMode     string
		adminIP             string
		wantMaintenanceMode bool
		wantAdminIPs        []string
	}{
		{
			name:                "メンテナンスモードON、単一IP",
			maintenanceMode:     "on",
			adminIP:             "192.168.1.1",
			wantMaintenanceMode: true,
			wantAdminIPs:        []string{"192.168.1.1"},
		},
		{
			name:                "メンテナンスモードON、複数IP",
			maintenanceMode:     "on",
			adminIP:             "192.168.1.1,10.0.0.1",
			wantMaintenanceMode: true,
			wantAdminIPs:        []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:                "メンテナンスモードOFF",
			maintenanceMode:     "off",
			adminIP:             "192.168.1.1",
			wantMaintenanceMode: false,
			wantAdminIPs:        []string{"192.168.1.1"},
		},
		{
			name:                "メンテナンスモード未設定",
			maintenanceMode:     "",
			adminIP:             "",
			wantMaintenanceMode: false,
			wantAdminIPs:        nil,
		},
		{
			name:                "メンテナンスモードON、管理者IP未設定",
			maintenanceMode:     "on",
			adminIP:             "",
			wantMaintenanceMode: true,
			wantAdminIPs:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			if tt.maintenanceMode != "" {
				_ = os.Setenv("MEWST_MAINTENANCE_MODE", tt.maintenanceMode)
			} else {
				_ = os.Unsetenv("MEWST_MAINTENANCE_MODE")
			}
			if tt.adminIP != "" {
				_ = os.Setenv("MEWST_ADMIN_IP", tt.adminIP)
			} else {
				_ = os.Unsetenv("MEWST_ADMIN_IP")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}

			if cfg.MaintenanceMode != tt.wantMaintenanceMode {
				t.Errorf("MaintenanceMode = %v、期待値 = %v", cfg.MaintenanceMode, tt.wantMaintenanceMode)
			}
			if !reflect.DeepEqual(cfg.AdminIPs, tt.wantAdminIPs) {
				t.Errorf("AdminIPs = %v、期待値 = %v", cfg.AdminIPs, tt.wantAdminIPs)
			}
		})
	}
}

// TestLoad_DisableRateLimitはDisableRateLimit設定のテスト
func TestLoad_DisableRateLimit(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"未設定", "", false},
		{"other", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			if tt.value != "" {
				_ = os.Setenv("MEWST_DISABLE_RATE_LIMIT", tt.value)
			} else {
				_ = os.Unsetenv("MEWST_DISABLE_RATE_LIMIT")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}

			if cfg.DisableRateLimit != tt.want {
				t.Errorf("DisableRateLimit = %v、期待値 = %v", cfg.DisableRateLimit, tt.want)
			}
		})
	}
}

// TestLoad_RailsAppURLはRailsAppURL設定のテスト
func TestLoad_RailsAppURL(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	railsURL := "http://localhost:3001"
	_ = os.Setenv("MEWST_RAILS_APP_URL", railsURL)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load()のエラー = %v", err)
	}

	if cfg.RailsAppURL != railsURL {
		t.Errorf("RailsAppURL = %v、期待値 = %v", cfg.RailsAppURL, railsURL)
	}
}

// TestParseAdminIPsはparseAdminIPs関数のテスト
func TestParseAdminIPs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "単一IP",
			input: "192.168.1.1",
			want:  []string{"192.168.1.1"},
		},
		{
			name:  "複数IP",
			input: "192.168.1.1,10.0.0.1",
			want:  []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:  "複数IPスペースあり",
			input: "192.168.1.1, 10.0.0.1, 172.16.0.1",
			want:  []string{"192.168.1.1", "10.0.0.1", "172.16.0.1"},
		},
		{
			name:  "空白のみの要素を除去",
			input: "192.168.1.1,  ,10.0.0.1",
			want:  []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:  "空文字列",
			input: "",
			want:  []string{},
		},
		{
			name:  "先頭と末尾の空白を除去",
			input: "  192.168.1.1  ",
			want:  []string{"192.168.1.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAdminIPs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAdminIPs(%q) = %v、期待値 = %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestLoad_SentryConfigはSentry環境変数の読み込みをテスト
func TestLoad_SentryConfig(t *testing.T) {
	tests := []struct {
		name                 string
		dsn                  string
		environment          string
		tracesSampleRate     string
		debug                string
		wantDSN              string
		wantEnvironment      string
		wantTracesSampleRate float64
		wantDebug            bool
	}{
		{
			name:                 "全て未設定 (Sentry無効化、EnvironmentはAPP_ENVにフォールバック)",
			dsn:                  "",
			environment:          "",
			tracesSampleRate:     "",
			debug:                "",
			wantDSN:              "",
			wantEnvironment:      "test", // setupTestEnvでAPP_ENV=testに設定済み
			wantTracesSampleRate: 0.5,
			wantDebug:            false,
		},
		{
			name:                 "全て設定",
			dsn:                  "https://example@o0.ingest.sentry.io/0",
			environment:          "prod",
			tracesSampleRate:     "0.25",
			debug:                "true",
			wantDSN:              "https://example@o0.ingest.sentry.io/0",
			wantEnvironment:      "prod",
			wantTracesSampleRate: 0.25,
			wantDebug:            true,
		},
		{
			name:                 "DSNのみ設定 (他はデフォルト)",
			dsn:                  "https://example@o0.ingest.sentry.io/0",
			environment:          "",
			tracesSampleRate:     "",
			debug:                "",
			wantDSN:              "https://example@o0.ingest.sentry.io/0",
			wantEnvironment:      "test",
			wantTracesSampleRate: 0.5,
			wantDebug:            false,
		},
		{
			name:                 "Debugがtrue以外の値はfalse扱い",
			dsn:                  "",
			environment:          "",
			tracesSampleRate:     "",
			debug:                "yes",
			wantDSN:              "",
			wantEnvironment:      "test",
			wantTracesSampleRate: 0.5,
			wantDebug:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			if tt.dsn != "" {
				_ = os.Setenv("MEWST_SENTRY_DSN", tt.dsn)
			}
			if tt.environment != "" {
				_ = os.Setenv("MEWST_SENTRY_ENVIRONMENT", tt.environment)
			}
			if tt.tracesSampleRate != "" {
				_ = os.Setenv("MEWST_SENTRY_TRACES_SAMPLE_RATE", tt.tracesSampleRate)
			}
			if tt.debug != "" {
				_ = os.Setenv("MEWST_SENTRY_DEBUG", tt.debug)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}

			if cfg.SentryDSN != tt.wantDSN {
				t.Errorf("SentryDSN = %q、期待値 = %q", cfg.SentryDSN, tt.wantDSN)
			}
			if cfg.SentryEnvironment != tt.wantEnvironment {
				t.Errorf("SentryEnvironment = %q、期待値 = %q", cfg.SentryEnvironment, tt.wantEnvironment)
			}
			if cfg.SentryTracesSampleRate != tt.wantTracesSampleRate {
				t.Errorf("SentryTracesSampleRate = %v、期待値 = %v", cfg.SentryTracesSampleRate, tt.wantTracesSampleRate)
			}
			if cfg.SentryDebug != tt.wantDebug {
				t.Errorf("SentryDebug = %v、期待値 = %v", cfg.SentryDebug, tt.wantDebug)
			}
		})
	}
}

// TestLoad_SentryEnvironment_FallbackToAppEnvはSentryEnvironmentがAPP_ENVにフォールバックすることをテスト
func TestLoad_SentryEnvironment_FallbackToAppEnv(t *testing.T) {
	tests := []struct {
		name    string
		appEnv  string
		wantEnv string
	}{
		{name: "devへフォールバック", appEnv: "dev", wantEnv: "dev"},
		{name: "testへフォールバック", appEnv: "test", wantEnv: "test"},
		{name: "prodへフォールバック", appEnv: "prod", wantEnv: "prod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			_ = os.Setenv("APP_ENV", tt.appEnv)
			if tt.appEnv == "prod" {
				setExportStorageEnv(t)
			}
			_ = os.Unsetenv("MEWST_SENTRY_ENVIRONMENT")

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}

			if cfg.SentryEnvironment != tt.wantEnv {
				t.Errorf("SentryEnvironment = %q、期待値 = %q", cfg.SentryEnvironment, tt.wantEnv)
			}
		})
	}
}

// TestParseSentryTracesSampleRateはparseSentryTracesSampleRate関数のテスト
func TestParseSentryTracesSampleRate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{name: "空文字列はデフォルト0.5", input: "", want: 0.5},
		{name: "下限0.0", input: "0.0", want: 0.0},
		{name: "中間値0.5", input: "0.5", want: 0.5},
		{name: "0.25を受理", input: "0.25", want: 0.25},
		{name: "上限1.0", input: "1.0", want: 1.0},
		{name: "範囲外 (負数) はデフォルト0.5", input: "-0.1", want: 0.5},
		{name: "範囲外 (1.0超過) はデフォルト0.5", input: "1.5", want: 0.5},
		{name: "パース不可文字列はデフォルト0.5", input: "abc", want: 0.5},
		{name: "整数表記も許容", input: "1", want: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSentryTracesSampleRate(tt.input)
			if got != tt.want {
				t.Errorf("parseSentryTracesSampleRate(%q) = %v、期待値 = %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestGetAssetVersionはGetAssetVersionメソッドのテスト
func TestGetAssetVersion(t *testing.T) {
	tests := []struct {
		name         string
		env          string
		assetVersion string
		wantStatic   bool
	}{
		{
			name:         "開発環境では動的値",
			env:          "dev",
			assetVersion: "abc123",
			wantStatic:   false,
		},
		{
			name:         "テスト環境ではGitハッシュ",
			env:          "test",
			assetVersion: "abc123",
			wantStatic:   true,
		},
		{
			name:         "本番環境ではGitハッシュ",
			env:          "prod",
			assetVersion: "abc123",
			wantStatic:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Env: tt.env, AssetVersion: tt.assetVersion}

			got1 := cfg.GetAssetVersion()
			got2 := cfg.GetAssetVersion()

			if tt.wantStatic {
				// 静的値 (AssetVersion) が返されるべき
				if got1 != tt.assetVersion {
					t.Errorf("GetAssetVersion() = %v、期待値 = %v", got1, tt.assetVersion)
				}
				if got1 != got2 {
					t.Errorf("GetAssetVersion()が同じ値を返さない: %vと%v", got1, got2)
				}
			} else {
				// 動的値 (タイムスタンプ) が返されるべき
				if got1 == "" {
					t.Error("GetAssetVersion() = 空文字列、非空を期待")
				}
			}
		})
	}
}

// GIT_REV環境変数が最優先され、7文字に短縮されることを検証する。
//
// Dokkuのデプロイ先には .gitが無くgitコマンドが失敗するため、GIT_REVを
// 使えるかどうかがSentryのrelease ("dev" 化の回避) を左右する。
func TestGetGitCommitHash(t *testing.T) {
	tests := []struct {
		name   string
		gitRev string
		want   string
	}{
		{
			name:   "フルSHAは7文字に短縮される",
			gitRev: "1234567890abcdef1234567890abcdef12345678",
			want:   "1234567",
		},
		{
			name:   "7文字以下ならそのまま返す",
			gitRev: "abc123",
			want:   "abc123",
		},
		{
			name:   "前後の空白は除去される",
			gitRev: "  1234567890abcdef  ",
			want:   "1234567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GIT_REV", tt.gitRev)

			got := getGitCommitHash()
			if got != tt.want {
				t.Errorf("getGitCommitHash() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

// TestS3Readinessはエクスポート用ストレージreadinessの契約テスト。
// 全項目未設定はdisabled、必須項目の完全設定はready、部分設定 (リージョンのみの
// 設定を含む) はinvalidになる。
func TestS3Readiness(t *testing.T) {
	t.Parallel()

	const (
		bucket    = "mewst-export-test"
		endpoint  = "https://example.r2.cloudflarestorage.com"
		accessKey = "test-access-key-id"
		secretKey = "test-secret-access-key"
	)

	tests := []struct {
		name string
		cfg  Config
		want S3Readiness
	}{
		{
			name: "全項目未設定: disabled",
			cfg:  Config{},
			want: S3ReadinessDisabled,
		},
		{
			name: "必須4項目設定: ready",
			cfg: Config{
				S3BucketName:      bucket,
				S3Endpoint:        endpoint,
				S3AccessKeyID:     accessKey,
				S3SecretAccessKey: secretKey,
			},
			want: S3ReadinessReady,
		},
		{
			name: "必須4項目 + リージョン設定: ready",
			cfg: Config{
				S3BucketName:      bucket,
				S3Endpoint:        endpoint,
				S3AccessKeyID:     accessKey,
				S3SecretAccessKey: secretKey,
				S3Region:          "apac",
			},
			want: S3ReadinessReady,
		},
		{
			name: "バケットのみ設定: invalid",
			cfg:  Config{S3BucketName: bucket},
			want: S3ReadinessInvalid,
		},
		{
			name: "シークレットのみ欠け: invalid",
			cfg: Config{
				S3BucketName:  bucket,
				S3Endpoint:    endpoint,
				S3AccessKeyID: accessKey,
			},
			want: S3ReadinessInvalid,
		},
		{
			name: "リージョンのみ設定: invalid",
			cfg:  Config{S3Region: "auto"},
			want: S3ReadinessInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.cfg.S3Readiness(); got != tt.want {
				t.Errorf("S3Readiness() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

// TestLoad_S3ConfigはMEWST_S3_* の読み込み契約を検証する。全項目未設定なら
// 起動でき (エクスポート無効)、完全設定でも起動でき (リージョンは "auto" に既定化)、
// 部分設定では起動に失敗する。
func TestLoad_S3Config(t *testing.T) {
	const (
		bucket    = "mewst-export-test"
		endpoint  = "https://example.r2.cloudflarestorage.com"
		accessKey = "test-access-key-id"
		secretKey = "test-secret-access-key"
	)

	tests := []struct {
		name          string
		env           map[string]string
		wantErr       bool
		wantReadiness S3Readiness
		wantRegion    string
	}{
		{
			name:          "全項目未設定: エクスポート無効のまま起動できる",
			env:           map[string]string{},
			wantErr:       false,
			wantReadiness: S3ReadinessDisabled,
			wantRegion:    "",
		},
		{
			name: "完全設定 (リージョンあり): 起動できる",
			env: map[string]string{
				"MEWST_S3_BUCKET_NAME":       bucket,
				"MEWST_S3_ENDPOINT":          endpoint,
				"MEWST_S3_ACCESS_KEY_ID":     accessKey,
				"MEWST_S3_SECRET_ACCESS_KEY": secretKey,
				"MEWST_S3_REGION":            "apac",
			},
			wantErr:       false,
			wantReadiness: S3ReadinessReady,
			wantRegion:    "apac",
		},
		{
			name: "完全設定 (リージョンなし): autoに既定化して起動できる",
			env: map[string]string{
				"MEWST_S3_BUCKET_NAME":       bucket,
				"MEWST_S3_ENDPOINT":          endpoint,
				"MEWST_S3_ACCESS_KEY_ID":     accessKey,
				"MEWST_S3_SECRET_ACCESS_KEY": secretKey,
			},
			wantErr:       false,
			wantReadiness: S3ReadinessReady,
			wantRegion:    "auto",
		},
		{
			name: "部分設定 (バケットのみ): 起動時エラー",
			env: map[string]string{
				"MEWST_S3_BUCKET_NAME": bucket,
			},
			wantErr: true,
		},
		{
			name: "部分設定 (シークレット欠け): 起動時エラー",
			env: map[string]string{
				"MEWST_S3_BUCKET_NAME":   bucket,
				"MEWST_S3_ENDPOINT":      endpoint,
				"MEWST_S3_ACCESS_KEY_ID": accessKey,
			},
			wantErr: true,
		},
		{
			name: "リージョンのみ設定: 構成ミスとして起動時エラー",
			env: map[string]string{
				"MEWST_S3_REGION": "auto",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			for key, value := range tt.env {
				_ = os.Setenv(key, value)
			}

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Fatal("MEWST_S3_*の設定が一部だけのとき、Load()がエラーを返すことを期待したが、nilだった")
				}
				return
			}

			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}
			if got := cfg.S3Readiness(); got != tt.wantReadiness {
				t.Errorf("S3Readiness() = %q、期待値 = %q", got, tt.wantReadiness)
			}
			if cfg.S3Region != tt.wantRegion {
				t.Errorf("S3Region = %q、期待値 = %q", cfg.S3Region, tt.wantRegion)
			}
		})
	}
}

// TestLoad_S3ConfigInProductionは、本番がエクスポート用ストレージを必須と
// することを検証する。エクスポートが全ユーザーに公開された今、MEWST_S3_* 無しで
// 起動した本番プロセスはエクスポート画面を全員に対して利用不可の画面で返すことに
// なるため、Loadはこれを拒否する。非本番では同じ未設定でも起動でき、ローカル実行
// とテストがバケットの資格情報を持たずに済む。
func TestLoad_S3ConfigInProduction(t *testing.T) {
	tests := []struct {
		name         string
		appEnv       string
		fullS3Config bool
		partialS3Env map[string]string
		wantErr      bool
	}{
		{
			name:    "本番・全項目未設定: 起動時エラー",
			appEnv:  "prod",
			wantErr: true,
		},
		{
			name:         "本番・完全設定: 起動できる",
			appEnv:       "prod",
			fullS3Config: true,
			wantErr:      false,
		},
		{
			name:         "本番・部分設定: 起動時エラー",
			appEnv:       "prod",
			partialS3Env: map[string]string{"MEWST_S3_BUCKET_NAME": "mewst-export-prod"},
			wantErr:      true,
		},
		{
			name:    "開発・全項目未設定: 起動できる",
			appEnv:  "dev",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			_ = os.Setenv("APP_ENV", tt.appEnv)
			if tt.fullS3Config {
				setExportStorageEnv(t)
			}
			for key, value := range tt.partialS3Env {
				_ = os.Setenv(key, value)
			}

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Fatal("本番でエクスポート用ストレージの設定が揃っていないとき、Load()がエラーを返すことを期待したが、nilだった")
				}
				return
			}

			if err != nil {
				t.Fatalf("Load()のエラー = %v", err)
			}
			if cfg.Env != tt.appEnv {
				t.Errorf("Env = %q、期待値 = %q", cfg.Env, tt.appEnv)
			}
		})
	}
}
