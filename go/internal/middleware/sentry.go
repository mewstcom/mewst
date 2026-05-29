package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"

	"github.com/mewstcom/mewst/go/internal/model"
)

// SentryTransaction は chi のルートパターンを Sentry のトランザクション名に設定するミドルウェア。
//
// sentryhttp ミドルウェアは初期化時に URL ベースのトランザクション名 (例: "GET /sign_in") を設定するが、
// 動的パスを含むエンドポイント (例: "/users/{id}") では URL の値そのものがトランザクション名になり、
// カーディナリティが爆発する。本ミドルウェアは chi のルートパターン (例: "/users/{id}") を
// トランザクション名に上書きすることで、Sentry の Performance 画面で一貫したグルーピングを実現する。
//
// chi のルートパターンは next.ServeHTTP が完了した時点で確定するため、defer 内で取得する。
// 本ミドルウェアは sentryhttp ミドルウェアの直後に登録すること。これにより本ミドルウェアの defer が
// sentryhttp の defer (transaction.Finish()) より先に走り、Finish 前に Name を確定できる。
//
// 前提: chi v5 は Mux.ServeHTTP の入口で *chi.Context (rctx) を作成し、r.WithContext で
// r.Context() に rctx の **ポインタ** を注入する。ルートマッチング時には同じ rctx の
// RoutePattern フィールドを直接書き換えるため、Request 構造体自体を入れ替えずに
// rctx の中身だけが mutate される。よって、最初に渡された r をそのまま defer でキャプチャしても、
// next.ServeHTTP 完了後の r.Context() から最新の RoutePattern を読み取れる。
// chi のメジャーバージョン更新で WithContext の扱いが変わった場合、本ミドルウェアは破綻し得るため
// TestSentryTransaction_SetsTransactionName で不変条件を担保する。
func SentryTransaction(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer setSentryTransactionName(r)
		next.ServeHTTP(w, r)
	})
}

// setSentryTransactionName は chi のルートパターンを Sentry のトランザクション名に反映する。
// ルートパターンが未確定な場合 (静的ファイル / chi にマッチしないパス) は何もしない。
//
// Performance トランザクション (transaction.Name) とエラーイベント (event.Transaction) の双方を更新する:
//   - Performance トランザクション側: sentry.TransactionFromContext(...).Name に書き込む。
//     transaction.Finish() 時に event.Transaction が transaction.Name から自動的に設定される。
//   - エラーイベント側: sentry-go v0.46 では Scope に SetTransaction メソッドがなく、
//     panic 由来の event.Transaction を自動で埋める仕組みがない。そのため
//     hub.Scope().AddEventProcessor で「event.Transaction が空ならルート名を入れる」プロセッサを
//     defer 時に登録する。sentryhttp の defer (recoverWithSentry) はこの defer より後に走るため、
//     プロセッサ登録 → 例外キャプチャ → ApplyToEvent でプロセッサ適用、の順になる。
func setSentryTransactionName(r *http.Request) {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return
	}
	pattern := rctx.RoutePattern()
	if pattern == "" {
		return
	}

	name := r.Method + " " + pattern

	if transaction := sentry.TransactionFromContext(r.Context()); transaction != nil {
		transaction.Name = name
		transaction.Source = sentry.SourceRoute
	}

	if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
		hub.Scope().AddEventProcessor(func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			if event.Transaction == "" {
				event.Transaction = name
			}
			return event
		})
	}
}

// sentryProfileFinder は SentryUserContext が依存する Profile 取得の最小インターフェース。
// `*repository.ProfileRepository` を直接型として受けると、テストで DB を立てる必要が出るため、
// FindByID のみを切り出した極狭インターフェースとして定義する。
// `*repository.ProfileRepository` はこのインターフェースを構造的に満たす。
type sentryProfileFinder interface {
	FindByID(ctx context.Context, id model.ProfileID) (*model.Profile, error)
}

// SentryUserContext は認証済みリクエストのユーザー情報を Sentry のスコープに反映するミドルウェア。
//
// `UserFromContext` / `ActorFromContext` で context から認証済みユーザーとアクターを取り出し、
// Atname を取得するために `profileFinder.FindByID(actor.ProfileID)` を呼んで Profile を取得する。
// 取得した情報を `hub.Scope().SetUser(...)` でセットすることで、認証済みリクエストで発生した
// エラーやパフォーマンストレースに User ID と Atname (= Sentry の Username) が紐付く。
//
// 挙動:
//   - 未認証リクエスト (UserFromContext が nil) では何もしない (Sentry の User スコープは未更新)
//   - Hub が context にない (sentryhttp を通っていない経路) でも何もしない
//   - Actor が context にない場合は ID のみセット (Atname は埋まらない)
//   - Profile 取得に失敗 / 未存在の場合も ID のみセット (Sentry メタ情報取得失敗で本来の処理を巻き込まない)
//   - すべて取得できた場合は ID + Username (= Atname) をセット
//
// 認証ミドルウェア (`RequireAuth` / `RequireNoAuth` / `SetUser`) の直後に登録すること。
type SentryUserContext struct {
	profileFinder sentryProfileFinder
}

// NewSentryUserContext は SentryUserContext ミドルウェアを生成する
func NewSentryUserContext(profileFinder sentryProfileFinder) *SentryUserContext {
	return &SentryUserContext{profileFinder: profileFinder}
}

// Middleware は SentryUserContext のミドルウェアを返す
func (m *SentryUserContext) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		user := UserFromContext(ctx)
		if user == nil {
			next.ServeHTTP(w, r)
			return
		}

		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			next.ServeHTTP(w, r)
			return
		}

		sentryUser := sentry.User{ID: user.ID.String()}

		if actor := ActorFromContext(ctx); actor != nil {
			profile, err := m.profileFinder.FindByID(ctx, actor.ProfileID)
			switch {
			case err != nil:
				// Sentry 用のメタ情報取得失敗で本来のリクエスト処理を巻き込まないため、ログだけ残して継続する。
				slog.WarnContext(ctx, "Sentry 用のプロフィール取得に失敗", "error", err, "user_id", user.ID.String())
			case profile == nil:
				// データ不整合や論理削除のタイミングで未存在になるケースは ID のみ紐付ける。
				slog.WarnContext(ctx, "Sentry 用のプロフィールが見つからない", "user_id", user.ID.String(), "profile_id", actor.ProfileID.String())
			default:
				sentryUser.Username = profile.Atname
			}
		}

		hub.Scope().SetUser(sentryUser)

		next.ServeHTTP(w, r)
	})
}
