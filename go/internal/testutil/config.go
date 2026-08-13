package testutil

import (
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
)

// NewTestConfig はテスト用の標準的な *config.Config を返す。
// 各 handler の setupTestHandler から重複した cfg 構築を排除するために利用する。
// TurnstileSiteKey は常にダミー値を設定する (使わない handler でも害はない) 。
func NewTestConfig(t testing.TB) *config.Config {
	t.Helper()

	return &config.Config{
		Env:              "test",
		Port:             "3000",
		Domain:           "localhost",
		CookieDomain:     "localhost",
		SessionSecure:    false,
		SessionHTTPOnly:  true,
		TurnstileSiteKey: "test-site-key",
	}
}
