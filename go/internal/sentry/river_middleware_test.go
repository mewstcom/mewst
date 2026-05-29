package sentry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/riverqueue/river/rivertype"
)

// riverFakeTransport は sentry.Transport の実装で、送信されたイベントを記録する。
// river_middleware_test 専用に閉じておくため、他テストの fakeTransport 名と被らない名前にする。
type riverFakeTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *riverFakeTransport) Configure(_ sentry.ClientOptions) {}

func (t *riverFakeTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *riverFakeTransport) Flush(_ time.Duration) bool { return true }

func (t *riverFakeTransport) FlushWithContext(_ context.Context) bool { return true }

func (t *riverFakeTransport) Close() {}

func (t *riverFakeTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*sentry.Event, len(t.events))
	copy(out, t.events)
	return out
}

// newRiverTestHub はテスト用に独立した Hub と Transport を返す。
// グローバル `sentry.Init` を呼ばないため t.Parallel() でも他テストと干渉しない。
func newRiverTestHub(t *testing.T) (*sentry.Hub, *riverFakeTransport) {
	t.Helper()

	transport := &riverFakeTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         "https://public@example.com/1",
		Transport:   transport,
		Environment: "test",
	})
	if err != nil {
		t.Fatalf("sentry.NewClient() error = %v", err)
	}
	return sentry.NewHub(client, sentry.NewScope()), transport
}

// callRiverMiddleware は RiverWorkerMiddleware() を経由して doInner を実行する。
// ctx にテスト用 Hub を bind することで、ミドルウェア内の cloneHubForJob が
// その Hub を Clone してテスト用 transport にイベントが届くようにする。
func callRiverMiddleware(t *testing.T, hub *sentry.Hub, job *rivertype.JobRow, doInner func(ctx context.Context) error) error {
	t.Helper()

	ctx := sentry.SetHubOnContext(context.Background(), hub)

	mw := RiverWorkerMiddleware()
	wm, ok := mw.(rivertype.WorkerMiddleware)
	if !ok {
		t.Fatalf("RiverWorkerMiddleware() は rivertype.WorkerMiddleware を満たすべき")
	}
	return wm.Work(ctx, job, doInner)
}

func TestRiverWorkerMiddleware_NoErrorDoesNotSendEvent(t *testing.T) {
	t.Parallel()

	hub, transport := newRiverTestHub(t)
	job := &rivertype.JobRow{ID: 1, Kind: "send_email_confirmation", Attempt: 1}

	called := false
	err := callRiverMiddleware(t, hub, job, func(ctx context.Context) error {
		called = true
		// ctx に bind された Hub が存在することを確認する
		if !sentry.HasHubOnContext(ctx) {
			t.Error("doInner の ctx に Sentry Hub が bind されているべき")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("doInner が呼ばれていない")
	}
	if events := transport.Events(); len(events) != 0 {
		t.Errorf("成功時は Sentry に送信されないはず: got %d events", len(events))
	}
}

func TestRiverWorkerMiddleware_ErrorCapturesEvent(t *testing.T) {
	t.Parallel()

	hub, transport := newRiverTestHub(t)
	job := &rivertype.JobRow{ID: 2, Kind: "send_email_confirmation", Attempt: 3}

	jobErr := errors.New("ジョブ実行に失敗")
	err := callRiverMiddleware(t, hub, job, func(_ context.Context) error {
		return jobErr
	})
	if !errors.Is(err, jobErr) {
		t.Fatalf("ミドルウェアは doInner のエラーをそのまま返すべき: got %v, want %v", err, jobErr)
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("失敗時は Sentry に 1 件送るべき: got %d events", len(events))
	}
	if got := events[0].Tags["job.kind"]; got != "send_email_confirmation" {
		t.Errorf("job.kind タグが期待と異なる: got %q, want %q", got, "send_email_confirmation")
	}
	if got := events[0].Tags["job.attempt"]; got != "3" {
		t.Errorf("job.attempt タグが期待と異なる: got %q, want %q", got, "3")
	}
}

func TestRiverWorkerMiddleware_DropsIgnorableErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "context.Canceledは送らない", err: context.Canceled},
		{name: "context.Canceledのラップは送らない", err: fmt.Errorf("wrap: %w", context.Canceled)},
		{name: "http.ErrAbortHandlerは送らない", err: http.ErrAbortHandler},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hub, transport := newRiverTestHub(t)
			job := &rivertype.JobRow{ID: 3, Kind: "send_email_confirmation", Attempt: 1}

			err := callRiverMiddleware(t, hub, job, func(_ context.Context) error {
				return tt.err
			})
			// ミドルウェアはエラー自体は返す (river にリトライ判断を委ねるため)
			if !errors.Is(err, tt.err) {
				t.Errorf("エラーはそのまま返すべき: got %v, want %v", err, tt.err)
			}
			// ただし Sentry には送らない
			if events := transport.Events(); len(events) != 0 {
				t.Errorf("無視対象のエラーは Sentry に送られないはず: got %d events", len(events))
			}
		})
	}
}

func TestRiverWorkerMiddleware_BindsHubToContext(t *testing.T) {
	t.Parallel()

	hub, transport := newRiverTestHub(t)
	job := &rivertype.JobRow{ID: 4, Kind: "send_email_confirmation", Attempt: 1}

	// doInner 内から sentryslog 経由でエラーログを出した場合のキャプチャ経路を再現する。
	// ここでは ctx 上の Hub から直接 CaptureException を呼んで、テスト用 transport に
	// イベントが届き、かつ Clone された Hub のタグも引き継がれることを担保する。
	err := callRiverMiddleware(t, hub, job, func(ctx context.Context) error {
		if h := sentry.GetHubFromContext(ctx); h != nil {
			h.CaptureException(errors.New("Worker 内部からのキャプチャ"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("ctx に bind された Hub からのキャプチャが届いていない: got %d events", len(events))
	}
	if got := events[0].Tags["job.kind"]; got != "send_email_confirmation" {
		t.Errorf("ctx 上の Hub にも job.kind タグが付くべき: got %q", got)
	}
}
