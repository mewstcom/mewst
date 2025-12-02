// Package session はセッション管理機能を提供します
package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/mewstcom/mewst/internal/config"
	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/repository"
)

// CookieName はRails版と共有するセッションクッキー名
const CookieName = "mewst_session_token"

// MaxAge はクッキーの有効期限（10年、Rails版と同じ）
const MaxAge = 10 * 365 * 24 * 60 * 60

// Manager はセッション管理を行う
type Manager struct {
	sessionRepo *repository.SessionRepository
	actorRepo   *repository.ActorRepository
	userRepo    *repository.UserRepository
	cfg         *config.Config
}

// NewManager は新しいManagerを作成する
func NewManager(
	sessionRepo *repository.SessionRepository,
	actorRepo *repository.ActorRepository,
	userRepo *repository.UserRepository,
	cfg *config.Config,
) *Manager {
	return &Manager{
		sessionRepo: sessionRepo,
		actorRepo:   actorRepo,
		userRepo:    userRepo,
		cfg:         cfg,
	}
}

// GenerateToken はRailsのhas_secure_token互換のトークンを生成する
// 24バイトのランダムデータをBase64 URL-safeエンコードした32文字の文字列を返す
func GenerateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetSessionToken はクッキーからセッショントークンを取得する
func (m *Manager) GetSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetCurrentUser は現在のログインユーザーを取得する
// セッションが存在しない、または無効な場合はnilを返す
func (m *Manager) GetCurrentUser(ctx context.Context, r *http.Request) (*model.User, error) {
	token := m.GetSessionToken(r)
	if token == "" {
		return nil, nil
	}

	// セッションを取得
	session, err := m.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	// セッションからアクターを取得
	actor, err := m.actorRepo.GetByID(ctx, session.ActorID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	// アクターからユーザーを取得
	user, err := m.userRepo.GetByID(ctx, actor.UserID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}

// GetCurrentActor は現在のログインアクターを取得する
// セッションが存在しない、または無効な場合はnilを返す
func (m *Manager) GetCurrentActor(ctx context.Context, r *http.Request) (*model.Actor, error) {
	token := m.GetSessionToken(r)
	if token == "" {
		return nil, nil
	}

	// セッションを取得
	session, err := m.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	// セッションからアクターを取得
	actor, err := m.actorRepo.GetByID(ctx, session.ActorID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return actor, nil
}

// SetSessionCookie はセッションクッキーを設定する
func (m *Manager) SetSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := m.cfg.SessionSecure
	// リバースプロキシ経由のHTTPS接続を検出
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		secure = true
	}

	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   secure,
		HttpOnly: m.cfg.SessionHTTPOnly,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   MaxAge,
	}
	http.SetCookie(w, cookie)
}

// DeleteSessionCookie はセッションクッキーを削除する
func (m *Manager) DeleteSessionCookie(w http.ResponseWriter, r *http.Request) {
	secure := m.cfg.SessionSecure
	// リバースプロキシ経由のHTTPS接続を検出
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		secure = true
	}

	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   secure,
		HttpOnly: m.cfg.SessionHTTPOnly,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // 即座に削除
	}
	http.SetCookie(w, cookie)
}

// IsLoggedIn はログイン済みかどうかを返す
func (m *Manager) IsLoggedIn(ctx context.Context, r *http.Request) bool {
	user, err := m.GetCurrentUser(ctx, r)
	return err == nil && user != nil
}
