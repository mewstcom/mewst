// Package session はセッション管理機能を提供します
package session

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// CookieName はRails版と共有するセッションクッキー名
const CookieName = "mewst_session_token"

// EmailConfirmationCookieName はメール確認IDを保存するクッキー名
const EmailConfirmationCookieName = "mewst_email_confirmation_id"

// MaxAge はクッキーの有効期限 (10年、Rails版と同じ)
const MaxAge = 10 * 365 * 24 * 60 * 60

// EmailConfirmationMaxAge はメール確認クッキーの有効期限 (15分)
const EmailConfirmationMaxAge = 15 * 60

// Manager はセッション管理を行う
type Manager struct {
	sessionRepo *repository.SessionRepository
	actorRepo   *repository.ActorRepository
	userRepo    *repository.UserRepository
	profileRepo *repository.ProfileRepository
	cfg         *config.Config
}

// NewManager は新しいManagerを作成する
func NewManager(
	sessionRepo *repository.SessionRepository,
	actorRepo *repository.ActorRepository,
	userRepo *repository.UserRepository,
	profileRepo *repository.ProfileRepository,
	cfg *config.Config,
) *Manager {
	return &Manager{
		sessionRepo: sessionRepo,
		actorRepo:   actorRepo,
		userRepo:    userRepo,
		profileRepo: profileRepo,
		cfg:         cfg,
	}
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

	session, err := m.sessionRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	actor, err := m.actorRepo.FindByID(ctx, session.ActorID)
	if err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, nil
	}

	user, err := m.userRepo.FindByID(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
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

	session, err := m.sessionRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	actor, err := m.actorRepo.FindByID(ctx, session.ActorID)
	if err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, nil
	}

	return actor, nil
}

// GetCurrentProfile returns the profile of the current logged-in actor.
// It returns nil when the session is missing or invalid.
//
// [Ja] 現在のログインアクターのプロフィールを取得する。
// セッションが存在しない、または無効な場合はnilを返す。
func (m *Manager) GetCurrentProfile(ctx context.Context, r *http.Request) (*model.Profile, error) {
	actor, err := m.GetCurrentActor(ctx, r)
	if err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, nil
	}

	profile, err := m.profileRepo.FindByID(ctx, actor.ProfileID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, nil
	}

	return profile, nil
}

// SetSessionCookie はセッションクッキーを設定する
func (m *Manager) SetSessionCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.SessionSecure,
		HttpOnly: m.cfg.SessionHTTPOnly,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   MaxAge,
	}
	http.SetCookie(w, cookie)
}

// DeleteSessionCookie はセッションクッキーを削除する
func (m *Manager) DeleteSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.SessionSecure,
		HttpOnly: m.cfg.SessionHTTPOnly,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

// IsLoggedIn はログイン済みかどうかを返す
func (m *Manager) IsLoggedIn(ctx context.Context, r *http.Request) bool {
	user, err := m.GetCurrentUser(ctx, r)
	return err == nil && user != nil
}

// SetEmailConfirmationID はメール確認IDをクッキーに保存する
func (m *Manager) SetEmailConfirmationID(w http.ResponseWriter, id model.EmailConfirmationID) {
	cookie := &http.Cookie{
		Name:     EmailConfirmationCookieName,
		Value:    id.String(),
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.SessionSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   EmailConfirmationMaxAge,
	}
	http.SetCookie(w, cookie)
}

// GetEmailConfirmationID はクッキーからメール確認IDを取得する。
// クッキー不在または値が不正な UUID の場合は ok=false を返す。
func (m *Manager) GetEmailConfirmationID(r *http.Request) (model.EmailConfirmationID, bool) {
	cookie, err := r.Cookie(EmailConfirmationCookieName)
	if err != nil {
		return model.EmailConfirmationID{}, false
	}
	parsed, err := uuid.Parse(cookie.Value)
	if err != nil {
		return model.EmailConfirmationID{}, false
	}
	return model.EmailConfirmationID(parsed), true
}

// DeleteEmailConfirmationID はメール確認IDクッキーを削除する
func (m *Manager) DeleteEmailConfirmationID(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     EmailConfirmationCookieName,
		Value:    "",
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.SessionSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}
