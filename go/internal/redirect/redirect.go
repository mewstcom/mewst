// Package redirect はリダイレクトURLのバリデーションを提供する
package redirect

import (
	"net/url"
	"strings"
)

// ValidateBackURL は back パラメータの値が安全かどうかを検証する。
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

// GetSafeRedirectURL は安全なリダイレクトURLを返す。
// backURL が無効な場合はデフォルトURL ("/") を返す。
func GetSafeRedirectURL(backURL string) string {
	if ValidateBackURL(backURL) {
		return backURL
	}
	return "/"
}

// AppendSafeBack は base に "?back=" として safe な backURL を付加した URL を返す。
// backURL が無効な場合は base のみを返す。
// back を伝搬するリンク・リダイレクト先を組み立てる際に、ValidateBackURL の呼び忘れを防ぐ目的で利用する。
func AppendSafeBack(base, backURL string) string {
	if !ValidateBackURL(backURL) {
		return base
	}
	return base + "?back=" + url.QueryEscape(backURL)
}
