package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
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

// stubProfileFinder は sentryProfileFinder インターフェースをテスト用に満たすスタブ。
// 返却 (profile / err) を保持し、FindByID 呼び出し回数を記録する。
type stubProfileFinder struct {
	profile *model.Profile
	err     error
	calls   int
}

func (s *stubProfileFinder) FindByID(_ context.Context, _ model.ProfileID) (*model.Profile, error) {
	s.calls++
	return s.profile, s.err
}

// runSentryUserContextRequest は SentryUserContext ミドルウェアの挙動検証用に
// テスト用 Hub と認証 context をセットアップしたチェーンでリクエストを実行し、
// Sentry へ送信されたイベント群を返す。handler 内で CaptureMessage を呼ぶことで、
// その時点でスコープに乗っている User 情報が event.User に反映される。
func runSentryUserContextRequest(t *testing.T, finder sentryProfileFinder, ctxMutator func(context.Context) context.Context) []*sentry.Event {
	t.Helper()

	hub, transport := newSentryTestHub(t)
	mw := NewSentryUserContext(finder)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// handler 内で CaptureMessage を呼ぶことで、ミドルウェアが SetUser した直後の Scope を
		// 使ったイベントが送信される。これで Sentry に渡される User 情報を検証できる。
		if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
			hub.CaptureMessage("test event")
		}
		w.WriteHeader(http.StatusOK)
	})

	chain := bindHubMiddleware(hub)(mw.Middleware(handler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if ctxMutator != nil {
		req = req.WithContext(ctxMutator(req.Context()))
	}
	rr := httptest.NewRecorder()
	chain.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが期待と異なる: got %d, want %d", rr.Code, http.StatusOK)
	}
	return transport.Events()
}

func newTestUser() *model.User {
	return &model.User{
		ID:    model.UserID(uuid.New()),
		Email: "user@example.com",
	}
}

func newTestActor() *model.Actor {
	return &model.Actor{
		ID:        model.ActorID(uuid.New()),
		UserID:    model.UserID(uuid.New()),
		ProfileID: model.ProfileID(uuid.New()),
	}
}

func TestSentryUserContext_SetsUser_WithAtname(t *testing.T) {
	t.Parallel()

	user := newTestUser()
	actor := newTestActor()
	finder := &stubProfileFinder{profile: &model.Profile{Atname: "alice"}}

	events := runSentryUserContextRequest(t, finder, func(ctx context.Context) context.Context {
		ctx = context.WithValue(ctx, userContextKey, user)
		ctx = context.WithValue(ctx, actorContextKey, actor)
		return ctx
	})

	if finder.calls != 1 {
		t.Fatalf("FindByID 呼び出し回数が期待と異なる: got %d, want 1", finder.calls)
	}
	if len(events) != 1 {
		t.Fatalf("送信イベント数が期待と異なる: got %d, want 1", len(events))
	}
	ev := events[0]
	if ev.User.ID != user.ID.String() {
		t.Errorf("User.ID が期待と異なる: got %q, want %q", ev.User.ID, user.ID.String())
	}
	if ev.User.Username != "alice" {
		t.Errorf("User.Username が期待と異なる: got %q, want %q", ev.User.Username, "alice")
	}
}

func TestSentryUserContext_SetsUserIDOnly_WhenActorMissing(t *testing.T) {
	t.Parallel()

	user := newTestUser()
	finder := &stubProfileFinder{}

	events := runSentryUserContextRequest(t, finder, func(ctx context.Context) context.Context {
		return context.WithValue(ctx, userContextKey, user)
	})

	if finder.calls != 0 {
		t.Errorf("Actor がない場合は FindByID を呼ばないはず: got %d", finder.calls)
	}
	if len(events) != 1 {
		t.Fatalf("送信イベント数が期待と異なる: got %d, want 1", len(events))
	}
	ev := events[0]
	if ev.User.ID != user.ID.String() {
		t.Errorf("User.ID が期待と異なる: got %q, want %q", ev.User.ID, user.ID.String())
	}
	if ev.User.Username != "" {
		t.Errorf("User.Username は空のはず: got %q", ev.User.Username)
	}
}

func TestSentryUserContext_SetsUserIDOnly_WhenProfileNotFound(t *testing.T) {
	t.Parallel()

	user := newTestUser()
	actor := newTestActor()
	finder := &stubProfileFinder{profile: nil, err: nil}

	events := runSentryUserContextRequest(t, finder, func(ctx context.Context) context.Context {
		ctx = context.WithValue(ctx, userContextKey, user)
		ctx = context.WithValue(ctx, actorContextKey, actor)
		return ctx
	})

	if finder.calls != 1 {
		t.Fatalf("FindByID 呼び出し回数が期待と異なる: got %d, want 1", finder.calls)
	}
	if len(events) != 1 {
		t.Fatalf("送信イベント数が期待と異なる: got %d, want 1", len(events))
	}
	ev := events[0]
	if ev.User.ID != user.ID.String() {
		t.Errorf("User.ID が期待と異なる: got %q, want %q", ev.User.ID, user.ID.String())
	}
	if ev.User.Username != "" {
		t.Errorf("Profile 未存在では Username は空のはず: got %q", ev.User.Username)
	}
}

func TestSentryUserContext_SetsUserIDOnly_WhenProfileFetchFails(t *testing.T) {
	t.Parallel()

	user := newTestUser()
	actor := newTestActor()
	finder := &stubProfileFinder{err: errors.New("DB エラー")}

	events := runSentryUserContextRequest(t, finder, func(ctx context.Context) context.Context {
		ctx = context.WithValue(ctx, userContextKey, user)
		ctx = context.WithValue(ctx, actorContextKey, actor)
		return ctx
	})

	if finder.calls != 1 {
		t.Fatalf("FindByID 呼び出し回数が期待と異なる: got %d, want 1", finder.calls)
	}
	if len(events) != 1 {
		t.Fatalf("送信イベント数が期待と異なる: got %d, want 1", len(events))
	}
	ev := events[0]
	if ev.User.ID != user.ID.String() {
		t.Errorf("User.ID が期待と異なる: got %q, want %q", ev.User.ID, user.ID.String())
	}
	if ev.User.Username != "" {
		t.Errorf("Profile 取得失敗時は Username は空のはず: got %q", ev.User.Username)
	}
}

func TestSentryUserContext_NoOp_WhenUnauthenticated(t *testing.T) {
	t.Parallel()

	finder := &stubProfileFinder{}

	events := runSentryUserContextRequest(t, finder, nil)

	if finder.calls != 0 {
		t.Errorf("未認証時は FindByID を呼ばないはず: got %d", finder.calls)
	}
	if len(events) != 1 {
		t.Fatalf("送信イベント数が期待と異なる: got %d, want 1", len(events))
	}
	ev := events[0]
	if ev.User.ID != "" || ev.User.Username != "" {
		t.Errorf("未認証時は User.ID / Username は空のはず: got ID=%q Username=%q", ev.User.ID, ev.User.Username)
	}
}

func TestSentryUserContext_NoOp_WhenHubMissing(t *testing.T) {
	t.Parallel()

	user := newTestUser()
	finder := &stubProfileFinder{}
	mw := NewSentryUserContext(finder)

	// Hub を context にバインドせずに直接ミドルウェアを呼び、panic せず handler に処理が渡ることを確認する。
	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), userContextKey, user))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("ステータスコードが期待と異なる: got %d, want %d", rr.Code, http.StatusNoContent)
	}
	if finder.calls != 0 {
		t.Errorf("Hub がない場合は FindByID を呼ばないはず: got %d", finder.calls)
	}
}
