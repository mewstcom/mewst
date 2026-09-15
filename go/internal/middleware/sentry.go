package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"

	"github.com/mewstcom/mewst/go/internal/model"
)

// SentryTransactionはchiのルートパターンをSentryのトランザクション名に設定するミドルウェア。
//
// sentryhttpミドルウェアは初期化時にURLベースのトランザクション名 (例: "GET /sign_in") を設定するが、
// 動的パスを含むエンドポイント (例: "/users/{id}") ではURLの値そのものがトランザクション名になり、
// カーディナリティが爆発する。本ミドルウェアはchiのルートパターン (例: "/users/{id}") を
// トランザクション名に上書きすることで、SentryのPerformance画面で一貫したグルーピングを実現する。
//
// chiのルートパターンはnext.ServeHTTPが完了した時点で確定するため、defer内で取得する。
// 本ミドルウェアはsentryhttpミドルウェアの直後に登録すること。これにより本ミドルウェアのdeferが
// sentryhttpのdefer (transaction.Finish()) より先に走り、Finish前にNameを確定できる。
//
// 前提: chi v5はMux.ServeHTTPの入口で *chi.Context (rctx) を作成し、r.WithContextで
// r.Context() にrctxの **ポインタ** を注入する。ルートマッチング時には同じrctxの
// RoutePatternフィールドを直接書き換えるため、Request構造体自体を入れ替えずに
// rctxの中身だけがmutateされる。よって、最初に渡されたrをそのままdeferでキャプチャしても、
// next.ServeHTTP完了後のr.Context() から最新のRoutePatternを読み取れる。
// chiのメジャーバージョン更新でWithContextの扱いが変わった場合、本ミドルウェアは破綻し得るため
// TestSentryTransaction_SetsTransactionNameで不変条件を担保する。
func SentryTransaction(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer setSentryTransactionName(r)
		next.ServeHTTP(w, r)
	})
}

// setSentryTransactionNameはchiのルートパターンをSentryのトランザクション名に反映する。
// ルートパターンが未確定な場合 (静的ファイル / chiにマッチしないパス) は何もしない。
//
// Performanceトランザクション (transaction.Name) とエラーイベント (event.Transaction) の双方を更新する:
//   - Performanceトランザクション側: sentry.TransactionFromContext(...).Nameに書き込む。
//     transaction.Finish() 時にevent.Transactionがtransaction.Nameから自動的に設定される。
//   - エラーイベント側: sentry-go v0.46ではScopeにSetTransactionメソッドがなく、
//     panic由来のevent.Transactionを自動で埋める仕組みがない。そのため
//     hub.Scope().AddEventProcessorで「event.Transactionが空ならルート名を入れる」プロセッサを
//     defer時に登録する。sentryhttpのdefer (recoverWithSentry) はこのdeferより後に走るため、
//     プロセッサ登録 → 例外キャプチャ → ApplyToEventでプロセッサ適用、の順になる。
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

// sentryProfileFinderはSentryUserContextが依存するProfile取得の最小インターフェース。
// `*repository.ProfileRepository` を直接型として受けると、テストでDBを立てる必要が出るため、
// FindByIDのみを切り出した極狭インターフェースとして定義する。
// `*repository.ProfileRepository` はこのインターフェースを構造的に満たす。
type sentryProfileFinder interface {
	FindByID(ctx context.Context, id model.ProfileID) (*model.Profile, error)
}

// SentryUserContextは認証済みリクエストのユーザー情報をSentryのスコープに反映するミドルウェア。
//
// `UserFromContext` / `ActorFromContext` でcontextから認証済みユーザーとアクターを取り出し、
// Atnameを取得するために `profileFinder.FindByID(actor.ProfileID)` を呼んでProfileを取得する。
// 取得した情報を `hub.Scope().SetUser(...)` でセットすることで、認証済みリクエストで発生した
// エラーやパフォーマンストレースにUser IDとAtname (= SentryのUsername) が紐付く。
//
// 挙動:
//   - 未認証リクエスト (UserFromContextがnil) では何もしない (SentryのUserスコープは未更新)
//   - Hubがcontextにない (sentryhttpを通っていない経路) でも何もしない
//   - Actorがcontextにない場合はIDのみセット (Atnameは埋まらない)
//   - Profile取得に失敗 / 未存在の場合もIDのみセット (Sentryメタ情報取得失敗で本来の処理を巻き込まない)
//   - すべて取得できた場合はID + Username (= Atname) をセット
//
// 認証ミドルウェア (`RequireAuth` / `RequireNoAuth` / `SetUser`) の直後に登録すること。
type SentryUserContext struct {
	profileFinder sentryProfileFinder
}

// NewSentryUserContextはSentryUserContextミドルウェアを生成する
func NewSentryUserContext(profileFinder sentryProfileFinder) *SentryUserContext {
	return &SentryUserContext{profileFinder: profileFinder}
}

// MiddlewareはSentryUserContextのミドルウェアを返す
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
				// Sentry用のメタ情報取得失敗で本来のリクエスト処理を巻き込まないため、ログだけ残して継続する。
				slog.WarnContext(ctx, "Sentry用のプロフィール取得に失敗", "error", err, "user_id", user.ID.String())
			case profile == nil:
				// データ不整合や論理削除のタイミングで未存在になるケースはIDのみ紐付ける。
				slog.WarnContext(ctx, "Sentry用のプロフィールが見つからない", "user_id", user.ID.String(), "profile_id", actor.ProfileID.String())
			default:
				sentryUser.Username = profile.Atname
			}
		}

		hub.Scope().SetUser(sentryUser)

		next.ServeHTTP(w, r)
	})
}
