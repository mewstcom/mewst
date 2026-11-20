package testutil

import (
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
)

// NewTestConfigはテスト用の標準的な *config.Configを返す。
// 各handlerのsetupTestHandlerから重複したcfg構築を排除するために利用する。
// TurnstileSiteKeyは常にダミー値を設定する (使わないhandlerでも害はない) 。
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
