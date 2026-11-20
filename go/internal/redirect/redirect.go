// Package redirectはリダイレクトURLのバリデーションを提供する
package redirect

import (
	"net/url"
	"strings"
)

// ValidateBackURLはbackパラメータの値が安全かどうかを検証する。
//
// オープンリダイレクト攻撃を防ぐため、以下のルールでバリデーションを行う:
// - 空文字は無効
// - "/" で始まらない場合は無効 (相対パスのみ許可)
// - "//" で始まる場合は無効 (プロトコル相対URL)
func ValidateBackURL(backURL string) bool {
	if backURL == "" {
		return false
	}
	if !strings.HasPrefix(backURL, "/") {
		return false
	}
	if strings.HasPrefix(backURL, "//") {
		return false
	}
	return true
}

// GetSafeRedirectURLは安全なリダイレクトURLを返す。
// backURLが無効な場合はデフォルトURL ("/") を返す。
func GetSafeRedirectURL(backURL string) string {
	if ValidateBackURL(backURL) {
		return backURL
	}
	return "/"
}

// AppendSafeBackはbaseに "?back=" としてsafeなbackURLを付加したURLを返す。
// backURLが無効な場合はbaseのみを返す。
// backを伝搬するリンク・リダイレクト先を組み立てる際に、ValidateBackURLの呼び忘れを防ぐ目的で利用する。
func AppendSafeBack(base, backURL string) string {
	if !ValidateBackURL(backURL) {
		return base
	}
	return base + "?back=" + url.QueryEscape(backURL)
}
