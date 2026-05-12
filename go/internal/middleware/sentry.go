package middleware

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
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
