package session

import (
	"context"
	"net/http"
	"net/url"
)

// FlashType はフラッシュメッセージのタイプを表す
type FlashType string

// フラッシュメッセージのタイプ定数
const (
	FlashSuccess FlashType = "success"
	FlashError   FlashType = "error"
	FlashInfo    FlashType = "info"
	FlashWarning FlashType = "warning"
)

// Flash はフラッシュメッセージを表す
type Flash struct {
	Type    FlashType
	Message string
}

// flashContextKey はコンテキストに保存するフラッシュメッセージのキー
type flashContextKey struct{}

// SetFlashToContext はコンテキストにフラッシュメッセージを設定する
func SetFlashToContext(ctx context.Context, flash *Flash) context.Context {
	return context.WithValue(ctx, flashContextKey{}, flash)
}

// GetFlashFromContext はコンテキストからフラッシュメッセージを取得する
func GetFlashFromContext(ctx context.Context) *Flash {
	flash, ok := ctx.Value(flashContextKey{}).(*Flash)
	if !ok {
		return nil
	}
	return flash
}

// FlashCookieName はフラッシュメッセージ用クッキー名
const FlashCookieName = "mewst_flash"

// FlashTypeCookieName はフラッシュメッセージタイプ用クッキー名
const FlashTypeCookieName = "mewst_flash_type"

// SetFlashCookie はフラッシュメッセージをクッキーに設定する
// フラッシュメッセージは次のリクエストでのみ表示される
func (m *Manager) SetFlashCookie(w http.ResponseWriter, r *http.Request, flashType FlashType, message string) {
	secure := m.cfg.SessionSecure
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		secure = true
	}

	// メッセージクッキー（URLエンコードして非ASCII文字を安全に保存）
	messageCookie := &http.Cookie{
		Name:     FlashCookieName,
		Value:    url.QueryEscape(message),
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60, // 60秒で期限切れ
	}
	http.SetCookie(w, messageCookie)

	// タイプクッキー
	typeCookie := &http.Cookie{
		Name:     FlashTypeCookieName,
		Value:    string(flashType),
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60, // 60秒で期限切れ
	}
	http.SetCookie(w, typeCookie)
}

// GetFlashFromCookie はクッキーからフラッシュメッセージを取得し、削除する
func (m *Manager) GetFlashFromCookie(w http.ResponseWriter, r *http.Request) *Flash {
	messageCookie, err := r.Cookie(FlashCookieName)
	if err != nil {
		return nil
	}

	typeCookie, err := r.Cookie(FlashTypeCookieName)
	if err != nil {
		return nil
	}

	// クッキーを削除
	m.deleteFlashCookie(w, r)

	// URLデコードしてメッセージを復元
	message, err := url.QueryUnescape(messageCookie.Value)
	if err != nil {
		message = messageCookie.Value
	}

	return &Flash{
		Type:    FlashType(typeCookie.Value),
		Message: message,
	}
}

// deleteFlashCookie はフラッシュメッセージクッキーを削除する
func (m *Manager) deleteFlashCookie(w http.ResponseWriter, r *http.Request) {
	secure := m.cfg.SessionSecure
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		secure = true
	}

	// メッセージクッキーを削除
	messageCookie := &http.Cookie{
		Name:     FlashCookieName,
		Value:    "",
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, messageCookie)

	// タイプクッキーを削除
	typeCookie := &http.Cookie{
		Name:     FlashTypeCookieName,
		Value:    "",
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, typeCookie)
}
