// Package templates はテンプレートのヘルパー関数を提供します
package templates

import (
	"context"

	"github.com/a-h/templ"
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

// Icon はアイコン名からSVGを返す（templ.Component対応）
// 可変長引数でクラス名を指定可能: Icon("name", "class1 class2")
func Icon(name string, class ...string) templ.Component {
	icons := map[string]string{
		"arrow-left":       `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256"><path d="M224,128a8,8,0,0,1-8,8H59.31l58.35,58.34a8,8,0,0,1-11.32,11.32l-72-72a8,8,0,0,1,0-11.32l72-72a8,8,0,0,1,11.32,11.32L59.31,120H216A8,8,0,0,1,224,128Z"></path></svg>`,
		"arrow-right":      `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256"><path d="M221.66,133.66l-72,72a8,8,0,0,1-11.32-11.32L196.69,136H40a8,8,0,0,1,0-16H196.69L138.34,61.66a8,8,0,0,1,11.32-11.32l72,72A8,8,0,0,1,221.66,133.66Z"></path></svg>`,
		"error":            `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256"><path d="M236.8,188.09,149.35,36.22h0a24.76,24.76,0,0,0-42.7,0L19.2,188.09a23.51,23.51,0,0,0,0,23.72A24.35,24.35,0,0,0,40.55,224h174.9a24.35,24.35,0,0,0,21.33-12.19A23.51,23.51,0,0,0,236.8,188.09ZM222.93,203.8a8.5,8.5,0,0,1-7.48,4.2H40.55a8.5,8.5,0,0,1-7.48-4.2,7.59,7.59,0,0,1,0-7.72L120.52,44.21a8.75,8.75,0,0,1,15,0l87.45,151.87A7.59,7.59,0,0,1,222.93,203.8ZM120,144V104a8,8,0,0,1,16,0v40a8,8,0,0,1-16,0Zm20,36a12,12,0,1,1-12-12A12,12,0,0,1,140,180Z"></path></svg>`,
		"info":             `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256"><path d="M128,24A104,104,0,1,0,232,128,104.11,104.11,0,0,0,128,24Zm0,192a88,88,0,1,1,88-88A88.1,88.1,0,0,1,128,216Zm-8-80V80a8,8,0,0,1,16,0v56a8,8,0,0,1-16,0Zm20,36a12,12,0,1,1-12-12A12,12,0,0,1,140,172Z"></path></svg>`,
		"logo":             `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 512 512"><path d="M161.443 354.988L131.898 325.444C125.631 319.179 121.062 308.139 121.062 299.271V245.218C121.062 241.913 118.8 236.061 116.578 233.616L73.4335 186.159C71.3708 183.89 71.5696 180.472 73.8013 178.443L81.7349 171.231C84.0014 169.171 87.4214 169.366 89.4506 171.597L132.595 219.054C138.44 225.483 142.708 236.529 142.708 245.217V299.27C142.708 302.398 144.999 307.931 147.204 310.137L168.458 331.389L198.376 239.143C213.724 252.991 234.053 261.421 256.354 261.421C278.654 261.421 298.983 252.991 314.331 239.142L348.469 344.399C365.321 396.358 333.621 440 279.031 440H233.673C182.835 440 151.856 402.144 161.442 354.988L161.443 354.988ZM307.567 75.9157C314.048 68.1393 326.705 72.7222 326.705 82.8441V174.805C326.705 213.676 295.214 245.187 256.353 245.187C217.493 245.187 186.002 213.682 186.002 174.805V82.8441C186.002 72.7222 198.659 68.1393 205.14 75.9157L228.952 104.49H283.753L307.567 75.9157Z"/></svg>`,
		"paper-plane-tilt": `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256"><path d="M227.32,28.68a16,16,0,0,0-15.66-4.08l-.15,0L19.57,82.84a16,16,0,0,0-2.49,29.8L102,154l41.3,84.87A15.86,15.86,0,0,0,157.74,248q.69,0,1.38-.06a15.88,15.88,0,0,0,14-11.51l58.2-191.94c0-.05,0-.1,0-.15A16,16,0,0,0,227.32,28.68ZM157.83,231.85l-.05.14,0-.07-40.06-82.3,48-48a8,8,0,0,0-11.31-11.31l-48,48L24.08,98.25l-.07,0,.14,0L216,40Z"></path></svg>`,
		"sign-in":          `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256"><path d="M141.66,133.66l-40,40a8,8,0,0,1-11.32-11.32L116.69,136H24a8,8,0,0,1,0-16h92.69L90.34,93.66a8,8,0,0,1,11.32-11.32l40,40A8,8,0,0,1,141.66,133.66ZM200,32H136a8,8,0,0,0,0,16h56V208H136a8,8,0,0,0,0,16h64a8,8,0,0,0,8-8V40A8,8,0,0,0,200,32Z"></path></svg>`,
		"success":          `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256"><path d="M173.66,98.34a8,8,0,0,1,0,11.32l-56,56a8,8,0,0,1-11.32,0l-24-24a8,8,0,0,1,11.32-11.32L112,148.69l50.34-50.35A8,8,0,0,1,173.66,98.34ZM232,128A104,104,0,1,1,128,24,104.11,104.11,0,0,1,232,128Zm-16,0a88,88,0,1,0-88,88A88.1,88.1,0,0,0,216,128Z"></path></svg>`,
		"warning":          `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256"><path d="M128,24A104,104,0,1,0,232,128,104.11,104.11,0,0,0,128,24Zm0,192a88,88,0,1,1,88-88A88.1,88.1,0,0,1,128,216Zm-8-80V80a8,8,0,0,1,16,0v56a8,8,0,0,1-16,0Zm20,36a12,12,0,1,1-12-12A12,12,0,0,1,140,172Z"></path></svg>`,
	}

	svg, ok := icons[name]
	if !ok {
		svg = icons["info"]
	}

	// クラス名が指定されている場合は、SVGタグに追加
	if len(class) > 0 && class[0] != "" {
		// <svg の直後にclass属性を挿入
		svg = `<svg class="` + class[0] + `" ` + svg[5:]
	}

	return templ.Raw(svg)
}
