package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// fakeTransport は sentry.Transport の実装で、SendEvent された Event をすべて記録する。
// テスト中の Sentry クライアントに注入することで、グローバル状態を介さずに送信内容を検証できる。
type fakeTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *fakeTransport) Configure(_ sentry.ClientOptions) {}

func (t *fakeTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *fakeTransport) Flush(_ time.Duration) bool { return true }

func (t *fakeTransport) FlushWithContext(_ context.Context) bool { return true }

func (t *fakeTransport) Close() {}

func (t *fakeTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*sentry.Event, len(t.events))
	copy(out, t.events)
	return out
}

// newSentryTestHub はテスト用に独立した Hub と fakeTransport を返す。
// グローバルな sentry.Init は呼ばないため、t.Parallel() でも他テストと干渉しない。
func newSentryTestHub(t *testing.T) (*sentry.Hub, *fakeTransport) {
	t.Helper()

	transport := &fakeTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		// 有効な DSN を渡さないと client がイベントを処理しないため、ダミー DSN を設定する
		Dsn:              "https://public@example.com/1",
		Transport:        transport,
		EnableTracing:    true,
		TracesSampleRate: 1.0,
		Environment:      "test",
	})
	if err != nil {
		t.Fatalf("sentry.NewClient() error = %v", err)
	}
	return sentry.NewHub(client, sentry.NewScope()), transport
}

// bindHubMiddleware はリクエストコンテキストにテスト用 Hub を埋め込むミドルウェア。
// sentryhttp は GetHubFromContext で Hub を取得するため、これを sentryhttp より前に挿入することで
// テスト用 Hub に Sentry イベントを集約できる。
func bindHubMiddleware(hub *sentry.Hub) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := sentry.SetHubOnContext(r.Context(), hub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestSentryTransaction_SetsTransactionName(t *testing.T) {
	t.Parallel()

	hub, transport := newSentryTestHub(t)

	r := chi.NewRouter()
	r.Use(bindHubMiddleware(hub))
	r.Use(sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle)
	r.Use(SentryTransaction)
	r.Get("/users/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが期待と異なる: got %d, want %d", rr.Code, http.StatusOK)
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("送信イベント数が期待と異なる: got %d, want 1", len(events))
	}

	tx := events[0]
	if tx.Type != "transaction" {
		t.Errorf("Type が期待と異なる: got %q, want %q", tx.Type, "transaction")
	}
	if tx.Transaction != "GET /users/{id}" {
		t.Errorf("Transaction 名が期待と異なる: got %q, want %q", tx.Transaction, "GET /users/{id}")
	}
	if tx.TransactionInfo == nil {
		t.Fatal("TransactionInfo が nil")
	}
	if tx.TransactionInfo.Source != sentry.SourceRoute {
		t.Errorf("TransactionInfo.Source が期待と異なる: got %q, want %q", tx.TransactionInfo.Source, sentry.SourceRoute)
	}
}

func TestSentryTransaction_PanicCaptured(t *testing.T) {
	t.Parallel()

	hub, transport := newSentryTestHub(t)

	// 本番の main.go と同じ chi 内ミドルウェア順序:
	//   Recoverer (outer) → sentryhttp (Repanic: true) → SentryTransaction → handler
	// この並びでは handler の panic を innermost の sentryhttp の defer がまず捕捉し、
	// Sentry に送信したあと Repanic: true で再 panic、再 panic が outer の Recoverer に到達して 500 を書く。
	// Recoverer を chi 内に入れた状態でも sentryhttp が panic を捕捉できることを確認する。
	r := chi.NewRouter()
	r.Use(bindHubMiddleware(hub))
	r.Use(chimiddleware.Recoverer)
	r.Use(sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle)
	r.Use(SentryTransaction)
	r.Get("/users/{id}/panic", func(_ http.ResponseWriter, _ *http.Request) {
		panic("意図的な panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42/panic", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("ステータスコードが期待と異なる: got %d, want %d", rr.Code, http.StatusInternalServerError)
	}

	events := transport.Events()
	if len(events) != 2 {
		t.Fatalf("送信イベント数が期待と異なる (エラーとトランザクションの 2 件): got %d, want 2", len(events))
	}

	var errorEvent, txEvent *sentry.Event
	for _, e := range events {
		if e.Type == "transaction" {
			txEvent = e
		} else {
			errorEvent = e
		}
	}

	if errorEvent == nil {
		t.Fatal("エラーイベントが送信されていない")
	}
	if errorEvent.Level != sentry.LevelFatal {
		t.Errorf("エラーイベントの Level が期待と異なる: got %q, want %q", errorEvent.Level, sentry.LevelFatal)
	}
	// scope.span 経由で error イベントの Transaction にもルートパターンが反映されることを担保する。
	// Sentry SDK のバージョンアップでこの自動紐付けが壊れた場合、ここで気づける。
	if errorEvent.Transaction != "GET /users/{id}/panic" {
		t.Errorf("エラーイベントの Transaction 名が期待と異なる: got %q, want %q", errorEvent.Transaction, "GET /users/{id}/panic")
	}

	if txEvent == nil {
		t.Fatal("トランザクションイベントが送信されていない")
	}
	if txEvent.Transaction != "GET /users/{id}/panic" {
		t.Errorf("トランザクション名が期待と異なる: got %q, want %q", txEvent.Transaction, "GET /users/{id}/panic")
	}
}

func TestSentryTransaction_NoRouteContext(t *testing.T) {
	t.Parallel()

	// chi の RouteContext を持たない素の http.Request で middleware を呼んでも panic しないことを確認する。
	// 静的ファイル配信や chi の外で動くケースを想定。
	handler := SentryTransaction(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが期待と異なる: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSentryTransaction_UnmatchedRoute(t *testing.T) {
	t.Parallel()

	hub, transport := newSentryTestHub(t)

	r := chi.NewRouter()
	r.Use(bindHubMiddleware(hub))
	r.Use(sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle)
	r.Use(SentryTransaction)
	r.Get("/sign_in", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 未登録パスへのリクエストは 404 になる。sentry-go の既定では
	// TraceIgnoreStatusCodes に [404] が含まれており、404 のトランザクションは送信されない。
	// SentryTransaction はルートパターンが空のため何もしない (sentryhttp のデフォルト名 + URL ソースのまま)
	// が、結局トランザクション自体がドロップされるため、本ミドルウェアは未マッチルートでノイズを増やさない。
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("ステータスコードが期待と異なる: got %d, want %d", rr.Code, http.StatusNotFound)
	}

	events := transport.Events()
	if len(events) != 0 {
		for i, e := range events {
			t.Logf("予期せぬイベント %d: type=%q transaction=%q", i, e.Type, e.Transaction)
		}
		t.Fatalf("未マッチルート (404) ではイベントが送信されないはず: got %d, want 0", len(events))
	}
}
