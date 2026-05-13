package sentry

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

// slogFakeTransport は sentry.Transport の実装で、送信されたイベントをすべて記録する。
// slog_handler_test 専用に閉じておくため、既存の sentry_test.go の構造体名と被らない名前にする。
type slogFakeTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *slogFakeTransport) Configure(_ sentry.ClientOptions) {}

func (t *slogFakeTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *slogFakeTransport) Flush(_ time.Duration) bool { return true }

func (t *slogFakeTransport) FlushWithContext(_ context.Context) bool { return true }

func (t *slogFakeTransport) Close() {}

func (t *slogFakeTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*sentry.Event, len(t.events))
	copy(out, t.events)
	return out
}

// newSlogTestHub はテスト用に独立した Hub と fakeTransport を返す。
// グローバルな sentry.Init は呼ばないため、t.Parallel() でも他テストと干渉しない。
func newSlogTestHub(t *testing.T) (*sentry.Hub, *slogFakeTransport) {
	t.Helper()

	transport := &slogFakeTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		// 有効な DSN を渡さないと client がイベントを処理しないため、ダミー DSN を設定する
		Dsn:         "https://public@example.com/1",
		Transport:   transport,
		Environment: "test",
	})
	if err != nil {
		t.Fatalf("sentry.NewClient() error = %v", err)
	}
	return sentry.NewHub(client, sentry.NewScope()), transport
}

// newSlogLogger は base ハンドラー (バッファ書き込み) と Sentry ハンドラーを合成した logger を返す。
// バッファをアサーションで確認することで、base ハンドラーに「常に」ログが届くことを検証できる。
func newSlogLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(NewSlogHandler(base)), &buf
}

func TestSlogHandler_ErrorIsCapturedToSentry(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)
	logger, buf := newSlogLogger()

	// context にテスト用 Hub を載せると、sentryslog の eventHandler がその Hub にイベントを投げる。
	ctx := sentry.SetHubOnContext(context.Background(), hub)
	logger.ErrorContext(ctx, "テストエラー", "key", "value")

	// base ハンドラーには通常通りログが書き込まれる
	if !strings.Contains(buf.String(), "テストエラー") {
		t.Errorf("base ハンドラーにエラーログが書かれていない: got %q", buf.String())
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("Sentry に送信されたイベント数が期待と異なる: got %d, want 1", len(events))
	}
	if events[0].Level != sentry.LevelError {
		t.Errorf("Sentry イベントの Level が期待と異なる: got %q, want %q", events[0].Level, sentry.LevelError)
	}
	if events[0].Message != "テストエラー" {
		t.Errorf("Sentry イベントの Message が期待と異なる: got %q, want %q", events[0].Message, "テストエラー")
	}
}

func TestSlogHandler_InfoIsNotCapturedToSentry(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)
	logger, buf := newSlogLogger()

	ctx := sentry.SetHubOnContext(context.Background(), hub)
	logger.InfoContext(ctx, "テスト情報", "key", "value")

	// base ハンドラーには Info ログが書き込まれる
	if !strings.Contains(buf.String(), "テスト情報") {
		t.Errorf("base ハンドラーに Info ログが書かれていない: got %q", buf.String())
	}

	// Sentry には送信されない (EventLevel に Info を含めていないため)
	events := transport.Events()
	if len(events) != 0 {
		t.Errorf("Info レベルでは Sentry に送信されないはず: got %d events", len(events))
	}
}

func TestSlogHandler_WarnIsNotCapturedToSentry(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)
	logger, buf := newSlogLogger()

	ctx := sentry.SetHubOnContext(context.Background(), hub)
	logger.WarnContext(ctx, "テスト警告")

	if !strings.Contains(buf.String(), "テスト警告") {
		t.Errorf("base ハンドラーに Warn ログが書かれていない: got %q", buf.String())
	}

	// Warn は EventLevel ([Error, Fatal]) に含まれないため Sentry には送らない。
	// 「警告ログは運用上のノイズなので Sentry に流さない」というポリシーを担保する。
	events := transport.Events()
	if len(events) != 0 {
		t.Errorf("Warn レベルでは Sentry に送信されないはず: got %d events", len(events))
	}
}

func TestSlogHandler_WithAttrs_PropagatesToBothHandlers(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)

	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	// `slog.With(...)` は内部で Handler.WithAttrs を呼ぶ。これが multiHandler 経由で
	// base / sentry の両方に伝播することを担保する。
	logger := slog.New(NewSlogHandler(base)).With("service", "test", "request_id", "req-123")

	ctx := sentry.SetHubOnContext(context.Background(), hub)
	logger.ErrorContext(ctx, "属性付きエラー")

	// base ハンドラーには WithAttrs で設定した属性が反映される
	output := buf.String()
	if !strings.Contains(output, `service=test`) {
		t.Errorf("base ハンドラーに service 属性が反映されていない: got %q", output)
	}
	if !strings.Contains(output, `request_id=req-123`) {
		t.Errorf("base ハンドラーに request_id 属性が反映されていない: got %q", output)
	}

	// Sentry にもイベントが届く
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("Sentry に送信されたイベント数が期待と異なる: got %d, want 1", len(events))
	}
	if events[0].Message != "属性付きエラー" {
		t.Errorf("Sentry イベントの Message が期待と異なる: got %q, want %q", events[0].Message, "属性付きエラー")
	}
}

func TestSlogHandler_HubFromContext_PrefersRequestHub(t *testing.T) {
	t.Parallel()

	// 2 つの独立した Hub を用意し、ctx に bind した Hub にだけイベントが届くことを担保する。
	// これにより「リクエストごとに別々の Hub にイベントが分離される」挙動 (sentryhttp 経由のリクエスト Hub) を再現する。
	hubA, transportA := newSlogTestHub(t)
	_, transportB := newSlogTestHub(t)

	logger, _ := newSlogLogger()

	// hubA のみを ctx に bind する。
	ctx := sentry.SetHubOnContext(context.Background(), hubA)
	logger.ErrorContext(ctx, "hubA に届くべきエラー")

	if len(transportA.Events()) != 1 {
		t.Errorf("hubA に紐付いた transport にエラーが届かなかった: got %d events", len(transportA.Events()))
	}
	if len(transportB.Events()) != 0 {
		t.Errorf("hubB は ctx に bind されていないため何も届かないはず: got %d events", len(transportB.Events()))
	}
}
