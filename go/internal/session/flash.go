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

// formErrorsContextKey はコンテキストに保存するフォームエラーのキー
type formErrorsContextKey struct{}

// FormErrors はフォームバリデーションエラーを表す
type FormErrors struct {
	Global []string            // フィールド横断のグローバルエラー
	Fields map[string][]string // フィールドごとのエラー
}

// NewFormErrors は新しいFormErrorsを作成する
func NewFormErrors() *FormErrors {
	return &FormErrors{
		Global: []string{},
		Fields: make(map[string][]string),
	}
}

// AddGlobalError はグローバルエラーを追加する
func (fe *FormErrors) AddGlobalError(message string) {
	fe.Global = append(fe.Global, message)
}

// AddFieldError はフィールドエラーを追加する
func (fe *FormErrors) AddFieldError(field, message string) {
	if fe.Fields == nil {
		fe.Fields = make(map[string][]string)
	}
	fe.Fields[field] = append(fe.Fields[field], message)
}

// HasErrors はエラーが存在するかを返す
func (fe *FormErrors) HasErrors() bool {
	return len(fe.Global) > 0 || len(fe.Fields) > 0
}

// HasFieldError は特定のフィールドにエラーがあるかを返す
func (fe *FormErrors) HasFieldError(field string) bool {
	_, exists := fe.Fields[field]
	return exists
}

// GetFieldErrors は特定のフィールドのエラーを返す
func (fe *FormErrors) GetFieldErrors(field string) []string {
	if fe.Fields == nil {
		return nil
	}
	return fe.Fields[field]
}

// FieldError はフィールドエラーを表す構造体（テンプレート用）
type FieldError struct {
	Field   string
	Message string
}

// FieldErrors はフィールドエラーを列挙可能な形式で取得する
func (fe *FormErrors) FieldErrors() []FieldError {
	var errors []FieldError
	for field, messages := range fe.Fields {
		for _, message := range messages {
			errors = append(errors, FieldError{
				Field:   field,
				Message: message,
			})
		}
	}
	return errors
}

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

// SetFormErrorsToContext はコンテキストにフォームエラーを設定する
func SetFormErrorsToContext(ctx context.Context, errors *FormErrors) context.Context {
	return context.WithValue(ctx, formErrorsContextKey{}, errors)
}

// GetFormErrorsFromContext はコンテキストからフォームエラーを取得する
func GetFormErrorsFromContext(ctx context.Context) *FormErrors {
	errors, ok := ctx.Value(formErrorsContextKey{}).(*FormErrors)
	if !ok {
		return nil
	}
	return errors
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
