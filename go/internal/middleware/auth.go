// Package middleware はHTTPミドルウェアを提供します
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
)

// contextKey はコンテキストに値を保存するための型
type contextKey string

const (
	// userContextKey はコンテキストにユーザー情報を保存するためのキー
	userContextKey contextKey = "user"
	// actorContextKey はコンテキストにアクター情報を保存するためのキー
	actorContextKey contextKey = "actor"
	// profileContextKey is the key for storing profile information in the context.
	// [Ja] profileContextKey はコンテキストにプロフィール情報を保存するためのキー
	profileContextKey contextKey = "profile"
)

// Auth は認証関連のミドルウェアを提供する
type Auth struct {
	sessionMgr *session.Manager
}

// NewAuth は新しいAuthミドルウェアを作成する
func NewAuth(sessionMgr *session.Manager) *Auth {
	return &Auth{
		sessionMgr: sessionMgr,
	}
}

// RequireAuth は認証が必要なルートを保護するミドルウェア
// 未認証の場合はログインページにリダイレクトする
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Resolve user / actor / profile in a single call so that
		// token -> session -> actor is looked up only once per request.
		// [Ja] user / actor / profile を 1 回の呼び出しでまとめて解決し、
		// 1 リクエストで token -> session -> actor の lookup が 3 回走らないようにする。
		auth, err := a.sessionMgr.GetCurrentAuth(ctx, r)
		if err != nil {
			slog.ErrorContext(ctx, "認証チェック中にエラーが発生", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if auth == nil {
			http.Redirect(w, r, "/sign_in", http.StatusFound)
			return
		}

		// Store the user, actor, and profile information in the context.
		// [Ja] コンテキストにユーザー・アクター・プロフィール情報を設定
		ctx = context.WithValue(ctx, userContextKey, auth.User)
		ctx = context.WithValue(ctx, actorContextKey, auth.Actor)
		ctx = context.WithValue(ctx, profileContextKey, auth.Profile)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireNoAuth は未認証が必要なルートを保護するミドルウェア
// 認証済みの場合はホームページにリダイレクトする
func (a *Auth) RequireNoAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		user, err := a.sessionMgr.GetCurrentUser(ctx, r)
		if err != nil {
			slog.ErrorContext(ctx, "認証チェック中にエラーが発生", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if user != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SetUser はコンテキストにユーザー情報を設定するミドルウェア
// RequireAuthとは異なり、認証チェックは行わず、ログインしていればユーザー情報を設定する
// 認証の有無に関わらずリクエストは処理される
func (a *Auth) SetUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		user, err := a.sessionMgr.GetCurrentUser(ctx, r)
		if err != nil {
			// エラーが発生してもリクエストは処理を継続
			slog.WarnContext(ctx, "ユーザー情報の取得に失敗", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		if user != nil {
			ctx = context.WithValue(ctx, userContextKey, user)

			actor, err := a.sessionMgr.GetCurrentActor(ctx, r)
			if err != nil {
				slog.WarnContext(ctx, "アクター情報の取得に失敗", "error", err)
			} else if actor != nil {
				ctx = context.WithValue(ctx, actorContextKey, actor)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext はコンテキストからユーザー情報を取得する
// ユーザーが設定されていない場合はnilを返す
func UserFromContext(ctx context.Context) *model.User {
	user, ok := ctx.Value(userContextKey).(*model.User)
	if !ok {
		return nil
	}
	return user
}

// ActorFromContext はコンテキストからアクター情報を取得する
// アクターが設定されていない場合はnilを返す
func ActorFromContext(ctx context.Context) *model.Actor {
	actor, ok := ctx.Value(actorContextKey).(*model.Actor)
	if !ok {
		return nil
	}
	return actor
}

// ProfileFromContext returns the profile information from the context.
// It returns nil when no profile is set.
//
// [Ja] ProfileFromContext はコンテキストからプロフィール情報を取得する。
// プロフィールが設定されていない場合は nil を返す。
func ProfileFromContext(ctx context.Context) *model.Profile {
	profile, ok := ctx.Value(profileContextKey).(*model.Profile)
	if !ok {
		return nil
	}
	return profile
}
