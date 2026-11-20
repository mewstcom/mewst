// Package i18nは国際化機能を提供します
package i18n

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// 翻訳ファイルを埋め込み
//
//go:embed locales/*.toml
var localesFS embed.FS

// サポートする言語
const (
	LangJa      = "ja"
	LangEn      = "en"
	DefaultLang = LangJa
)

// contextキーの型
type contextKey string

const (
	localeContextKey    contextKey = "locale"
	localizerContextKey contextKey = "localizer"
)

// SupportedLangsはアプリケーションが提供するすべての言語を、バンドルが
// 読み込む順に並べたもの。リクエストの外でロケールを読む呼び出し元 (開発用の
// シードは名簿から読む) が、一覧の写しではなく、実際に翻訳ファイルを持つ言語と
// 突き合わせられるよう公開している。
var SupportedLangs = []string{LangJa, LangEn}

// グローバルなバンドル
var bundle *i18n.Bundle

// initでlocalesディレクトリから全ての翻訳ファイルを読み込む
func init() {
	// 日本語をデフォルト言語として設定
	bundle = i18n.NewBundle(language.Japanese)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// 翻訳ファイルを読み込み
	for _, lang := range SupportedLangs {
		data, err := localesFS.ReadFile(fmt.Sprintf("locales/%s.toml", lang))
		if err != nil {
			continue
		}

		bundle.MustParseMessageFileBytes(data, fmt.Sprintf("%s.toml", lang))
	}
}

// Tは翻訳関数 (テンプレートやGoコードから呼び出される)
func T(ctx context.Context, messageID string, templateData ...map[string]any) string {
	localizer := GetLocalizer(ctx)
	if localizer == nil {
		return messageID
	}

	config := &i18n.LocalizeConfig{
		MessageID: messageID,
	}

	// テンプレートデータがある場合は設定
	if len(templateData) > 0 && templateData[0] != nil {
		config.TemplateData = templateData[0]

		// Countが含まれている場合は複数形処理を有効にする
		if count, ok := templateData[0]["Count"].(int32); ok {
			config.PluralCount = int(count)
		} else if count, ok := templateData[0]["Count"].(int); ok {
			config.PluralCount = count
		}
	}

	message, err := localizer.Localize(config)
	if err != nil {
		// 翻訳が見つからない場合はメッセージIDを返す
		return messageID
	}

	return message
}

// GetLocaleはコンテキストから言語設定を取得する
func GetLocale(ctx context.Context) string {
	if locale, ok := ctx.Value(localeContextKey).(string); ok {
		return locale
	}
	return DefaultLang
}

// SetLocaleはコンテキストに言語設定を保存する
func SetLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeContextKey, locale)
}

// GetLocalizerはコンテキストからLocalizerを取得する
func GetLocalizer(ctx context.Context) *i18n.Localizer {
	if localizer, ok := ctx.Value(localizerContextKey).(*i18n.Localizer); ok {
		return localizer
	}
	// Localizerがない場合は作成
	locale := GetLocale(ctx)
	return i18n.NewLocalizer(bundle, locale)
}

// SetLocalizerはコンテキストにLocalizerを保存する
func SetLocalizer(ctx context.Context, localizer *i18n.Localizer) context.Context {
	return context.WithValue(ctx, localizerContextKey, localizer)
}

// DetectLanguageはリクエストのAccept-Languageヘッダーから言語を検出する
func DetectLanguage(r *http.Request) string {
	// Accept-Languageヘッダーから取得
	acceptLang := r.Header.Get("Accept-Language")
	if strings.Contains(acceptLang, "ja") {
		return LangJa
	}
	// jaが含まれていない場合のみenをチェック
	if strings.Contains(acceptLang, "en") {
		return LangEn
	}

	// デフォルトは日本語
	return DefaultLang
}

// NewLocalizerは指定されたロケールのLocalizerを作成する
func NewLocalizer(locale string) *i18n.Localizer {
	return i18n.NewLocalizer(bundle, locale)
}

// MiddlewareはI18nミドルウェアを提供する
// Accept-Languageヘッダーから言語を決定し、ロケールとLocalizerをcontextにセットする
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := DetectLanguage(r)
		localizer := i18n.NewLocalizer(bundle, locale)

		ctx := SetLocale(r.Context(), locale)
		ctx = SetLocalizer(ctx, localizer)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
