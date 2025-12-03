// Package templates はテンプレートのヘルパー関数を提供します
package templates

import (
	"context"

	"github.com/mewstcom/mewst/internal/config"
	"github.com/mewstcom/mewst/internal/i18n"
)

// configContextKey はコンテキストに保存する設定のキー
type configContextKey struct{}

// WithLocale はコンテキストにロケールを設定する
// i18nパッケージのSetLocaleに委譲する
func WithLocale(ctx context.Context, locale string) context.Context {
	return i18n.SetLocale(ctx, locale)
}

// Locale はコンテキストからロケールを取得する
// i18nパッケージのGetLocaleに委譲する
func Locale(ctx context.Context) string {
	return i18n.GetLocale(ctx)
}

// WithConfig はコンテキストに設定を設定する
func WithConfig(ctx context.Context, cfg *config.Config) context.Context {
	return context.WithValue(ctx, configContextKey{}, cfg)
}

// GetConfig はコンテキストから設定を取得する
func GetConfig(ctx context.Context) *config.Config {
	cfg, ok := ctx.Value(configContextKey{}).(*config.Config)
	if !ok {
		return nil
	}
	return cfg
}

// T は翻訳を取得する
// i18nパッケージに委譲する
func T(ctx context.Context, messageID string, args ...any) string {
	// argsがmap[string]anyの場合はそのまま渡す
	if len(args) > 0 {
		if templateData, ok := args[0].(map[string]any); ok {
			return i18n.T(ctx, messageID, templateData)
		}
	}
	return i18n.T(ctx, messageID)
}
