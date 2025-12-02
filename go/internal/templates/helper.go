// Package templates はテンプレートのヘルパー関数を提供します
package templates

import (
	"context"

	"github.com/mewstcom/mewst/internal/config"
)

// localeContextKey はコンテキストに保存するロケールのキー
type localeContextKey struct{}

// configContextKey はコンテキストに保存する設定のキー
type configContextKey struct{}

// WithLocale はコンテキストにロケールを設定する
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeContextKey{}, locale)
}

// Locale はコンテキストからロケールを取得する
func Locale(ctx context.Context) string {
	locale, ok := ctx.Value(localeContextKey{}).(string)
	if !ok || locale == "" {
		return "ja" // デフォルトは日本語
	}
	return locale
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
// i18nパッケージ実装後に本格的な翻訳に置き換えられる
// 現時点ではメッセージIDをそのまま返す（フォールバック動作）
func T(ctx context.Context, messageID string, args ...any) string {
	// TODO: i18nパッケージ実装後に置き換え
	// return i18n.T(ctx, messageID, args...)

	// 仮実装: 基本的な翻訳マップ
	translations := map[string]map[string]string{
		"ja": {
			// メタ
			"meta.title.sign_in.new": "ログイン",

			// フォームラベル
			"forms.attributes.session_form.email":    "メールアドレス",
			"forms.attributes.session_form.password": "パスワード",

			// 動詞
			"verbs.sign_in": "ログインする",

			// メッセージ
			"messages.sign_in.dont_have_an_account": "アカウント持ってない？",

			// リンク
			"meta.title.sign_up.new":          "アカウント作成",
			"meta.title.password_resets.new":  "パスワード忘れた？",
			"messages.authentication.sign_in": "ログインしました",

			// エラー
			"nouns.error": "エラー",
			"forms.errors.session_form.unauthenticated": "ログインに失敗しました。メールアドレスかパスワードが間違っています",

			// Turnstile
			"errors.turnstile_verification_failed": "ロボット検証に失敗しました。もう一度お試しください",
		},
		"en": {
			// メタ
			"meta.title.sign_in.new": "Sign in",

			// フォームラベル
			"forms.attributes.session_form.email":    "Email",
			"forms.attributes.session_form.password": "Password",

			// 動詞
			"verbs.sign_in": "Sign in",

			// メッセージ
			"messages.sign_in.dont_have_an_account": "Don't have an account?",

			// リンク
			"meta.title.sign_up.new":          "Sign up",
			"meta.title.password_resets.new":  "Forgot your password?",
			"messages.authentication.sign_in": "Signed in successfully.",

			// エラー
			"nouns.error": "Error",
			"forms.errors.session_form.unauthenticated": "Login failed. Email or password is incorrect",

			// Turnstile
			"errors.turnstile_verification_failed": "Robot verification failed. Please try again",
		},
	}

	locale := Locale(ctx)
	if trans, ok := translations[locale]; ok {
		if msg, ok := trans[messageID]; ok {
			return msg
		}
	}

	// フォールバック: メッセージIDを返す
	return messageID
}
