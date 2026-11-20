// Package middlewareはHTTPミドルウェアを提供します
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
)

// contextKeyはコンテキストに値を保存するための型
type contextKey string

const (
	// userContextKeyはコンテキストにユーザー情報を保存するためのキー
	userContextKey contextKey = "user"
	// actorContextKeyはコンテキストにアクター情報を保存するためのキー
	actorContextKey contextKey = "actor"
	// profileContextKeyはコンテキストにプロフィール情報を保存するためのキー
	profileContextKey contextKey = "profile"
)

// Authは認証関連のミドルウェアを提供する
type Auth struct {
	sessionMgr *session.Manager
}

// NewAuthは新しいAuthミドルウェアを作成する
func NewAuth(sessionMgr *session.Manager) *Auth {
	return &Auth{
		sessionMgr: sessionMgr,
	}
}

// RequireAuthは認証が必要なルートを保護するミドルウェア
// 未認証の場合はログインページにリダイレクトする
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// user / actor / profileを1回の呼び出しでまとめて解決し、
		// 1リクエストでtoken -> session -> actorのlookupが3回走らないようにする。
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

		// コンテキストにユーザー・アクター・プロフィール情報を設定
		ctx = SetUserToContext(ctx, auth.User)
		ctx = SetActorToContext(ctx, auth.Actor)
		ctx = SetProfileToContext(ctx, auth.Profile)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireNoAuthは未認証が必要なルートを保護するミドルウェア
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

// SetUserはコンテキストにユーザー情報を設定するミドルウェア
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
			ctx = SetUserToContext(ctx, user)

			actor, err := a.sessionMgr.GetCurrentActor(ctx, r)
			if err != nil {
				slog.WarnContext(ctx, "アクター情報の取得に失敗", "error", err)
			} else if actor != nil {
				ctx = SetActorToContext(ctx, actor)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SetUserToContextはコンテキストにユーザー情報を設定する。本番ではRequireAuthと
// SetUserが使い、ハンドラーテストはログイン中ユーザーを注入するために再利用する
// (SetProfileToContextに倣う)。
func SetUserToContext(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContextはコンテキストからユーザー情報を取得する
// ユーザーが設定されていない場合はnilを返す
func UserFromContext(ctx context.Context) *model.User {
	user, ok := ctx.Value(userContextKey).(*model.User)
	if !ok {
		return nil
	}
	return user
}

// SetActorToContextはコンテキストにアクター情報を設定する。本番では
// RequireAuthとSetUserが使い、ハンドラーテストはログイン中のアクターを注入する
// ために再利用する (SetUserToContext / SetProfileToContextに倣う)。
func SetActorToContext(ctx context.Context, actor *model.Actor) context.Context {
	return context.WithValue(ctx, actorContextKey, actor)
}

// ActorFromContextはコンテキストからアクター情報を取得する
// アクターが設定されていない場合はnilを返す
func ActorFromContext(ctx context.Context) *model.Actor {
	actor, ok := ctx.Value(actorContextKey).(*model.Actor)
	if !ok {
		return nil
	}
	return actor
}

// SetProfileToContextはコンテキストにプロフィール情報を設定する。
// 本番ではRequireAuthが使い、ハンドラーテストはプロフィールを注入するために
// 再利用する (SetCSRFTokenToContextに倣う)。
func SetProfileToContext(ctx context.Context, profile *model.Profile) context.Context {
	return context.WithValue(ctx, profileContextKey, profile)
}

// ProfileFromContextはコンテキストからプロフィール情報を取得する。
// プロフィールが設定されていない場合はnilを返す。
func ProfileFromContext(ctx context.Context) *model.Profile {
	profile, ok := ctx.Value(profileContextKey).(*model.Profile)
	if !ok {
		return nil
	}
	return profile
}
