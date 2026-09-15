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

// riverFakeTransportはsentry.Transportの実装で、送信されたイベントを記録する。
// river_middleware_test専用に閉じておくため、他テストのfakeTransport名と被らない名前にする。
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

// newRiverTestHubはテスト用に独立したHubとTransportを返す。
// グローバル `sentry.Init` を呼ばないためt.Parallel() でも他テストと干渉しない。
func newRiverTestHub(t *testing.T) (*sentry.Hub, *riverFakeTransport) {
	t.Helper()

	transport := &riverFakeTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         "https://public@example.com/1",
		Transport:   transport,
		Environment: "test",
	})
	if err != nil {
		t.Fatalf("sentry.NewClient()のエラー = %v", err)
	}
	return sentry.NewHub(client, sentry.NewScope()), transport
}

// callRiverMiddlewareはRiverWorkerMiddleware() を経由してdoInnerを実行する。
// ctxにテスト用Hubをbindすることで、ミドルウェア内のcloneHubForJobが
// そのHubをCloneしてテスト用transportにイベントが届くようにする。
func callRiverMiddleware(t *testing.T, hub *sentry.Hub, job *rivertype.JobRow, doInner func(ctx context.Context) error) error {
	t.Helper()

	ctx := sentry.SetHubOnContext(context.Background(), hub)

	mw := RiverWorkerMiddleware()
	wm, ok := mw.(rivertype.WorkerMiddleware)
	if !ok {
		t.Fatalf("RiverWorkerMiddleware() はrivertype.WorkerMiddlewareを満たすべき")
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
		// ctxにbindされたHubが存在することを確認する
		if !sentry.HasHubOnContext(ctx) {
			t.Error("doInnerのctxにSentry Hubがbindされているべき")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if !called {
		t.Fatal("doInnerが呼ばれていない")
	}
	if events := transport.Events(); len(events) != 0 {
		t.Errorf("成功時はSentryに送信されないはず: 実測値 = %d件のイベント", len(events))
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
		t.Fatalf("ミドルウェアはdoInnerのエラーをそのまま返すべき: 実測値 = %v、期待値 = %v", err, jobErr)
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("失敗時はSentryに1件送るべき: 実測値 = %d件のイベント", len(events))
	}
	if got := events[0].Tags["job.kind"]; got != "send_email_confirmation" {
		t.Errorf("job.kindタグが期待と異なる: 実測値 = %q、期待値 = %q", got, "send_email_confirmation")
	}
	if got := events[0].Tags["job.attempt"]; got != "3" {
		t.Errorf("job.attemptタグが期待と異なる: 実測値 = %q、期待値 = %q", got, "3")
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
			// ミドルウェアはエラー自体は返す (riverにリトライ判断を委ねるため)
			if !errors.Is(err, tt.err) {
				t.Errorf("エラーはそのまま返すべき: 実測値 = %v、期待値 = %v", err, tt.err)
			}
			// ただしSentryには送らない
			if events := transport.Events(); len(events) != 0 {
				t.Errorf("無視対象のエラーはSentryに送られないはず: 実測値 = %d件のイベント", len(events))
			}
		})
	}
}

func TestRiverWorkerMiddleware_BindsHubToContext(t *testing.T) {
	t.Parallel()

	hub, transport := newRiverTestHub(t)
	job := &rivertype.JobRow{ID: 4, Kind: "send_email_confirmation", Attempt: 1}

	// doInner内からsentryslog経由でエラーログを出した場合のキャプチャ経路を再現する。
	// ここではctx上のHubから直接CaptureExceptionを呼んで、テスト用transportに
	// イベントが届き、かつCloneされたHubのタグも引き継がれることを担保する。
	err := callRiverMiddleware(t, hub, job, func(ctx context.Context) error {
		if h := sentry.GetHubFromContext(ctx); h != nil {
			h.CaptureException(errors.New("Worker 内部からのキャプチャ"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("ctxにbindされたHubからのキャプチャが届いていない: 実測値 = %d件のイベント", len(events))
	}
	if got := events[0].Tags["job.kind"]; got != "send_email_confirmation" {
		t.Errorf("ctx上のHubにもjob.kindタグが付くべき: 実測値 = %q", got)
	}
}
