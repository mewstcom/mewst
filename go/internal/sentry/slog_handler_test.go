package sentry

import (
	"bytes"
	"context"
	"errors"
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

// newSlogTestHub returns an isolated Hub and fake transport for a test.
//
// [Ja] テストごとに独立した Hub と fake transport を返す。
func newSlogTestHub(t *testing.T) (*sentry.Hub, *slogFakeTransport) {
	t.Helper()
	return newSlogTestHubWithBeforeSend(t, nil)
}

// newSlogTestHubWithBeforeSend returns an isolated test Hub configured with
// the supplied BeforeSend callback. It never calls the global sentry.Init, so
// parallel tests cannot interfere through global Sentry state.
//
// [Ja] 指定された BeforeSend callback を設定した独立テスト用 Hub を返す。
// グローバルな sentry.Init は呼ばないため、並行テスト間で Sentry のグローバル状態を
// 介した干渉は起きない。
func newSlogTestHubWithBeforeSend(
	t *testing.T,
	beforeSend func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event,
) (*sentry.Hub, *slogFakeTransport) {
	t.Helper()

	transport := &slogFakeTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		// 有効な DSN を渡さないと client がイベントを処理しないため、ダミー DSN を設定する
		Dsn:         "https://public@example.com/1",
		Transport:   transport,
		Environment: "test",
		BeforeSend:  beforeSend,
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

	// Putting a test hub on the context makes the Sentry handler capture the
	// event onto that hub.
	//
	// [Ja] context にテスト用 Hub を載せると、Sentry ハンドラーがその Hub に
	// event をキャプチャする。
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
	if events[0].Logger != "slog" {
		t.Errorf("Sentry イベントの Logger が期待と異なる: got %q, want %q", events[0].Logger, "slog")
	}
	if got := events[0].Tags["key"]; got != "value" {
		t.Errorf("Sentry イベントの key タグが期待と異なる: got %q, want %q", got, "value")
	}
	if len(events[0].Exception) != 0 {
		t.Errorf("error 属性がないログには Exception がないはず: got %+v", events[0].Exception)
	}
}

func TestSlogHandler_ErrorAttributeIsCapturedAsException(t *testing.T) {
	t.Parallel()

	var capturedHint *sentry.EventHint
	hub, transport := newSlogTestHubWithBeforeSend(
		t,
		func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			capturedHint = hint
			return event
		},
	)
	logger, _ := newSlogLogger()
	loggedErr := errors.New("Sentry テスト例外")
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	logger.ErrorContext(ctx, "例外付きエラー", "error", loggedErr, "request_id", "req-123")

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("Sentry に送信されたイベント数が期待と異なる: got %d, want 1", len(events))
	}
	if len(events[0].Exception) != 1 {
		t.Fatalf("Sentry イベントの Exception 数が期待と異なる: got %d, want 1", len(events[0].Exception))
	}
	if got := events[0].Exception[0].Value; got != loggedErr.Error() {
		t.Errorf("Sentry Exception の値が期待と異なる: got %q, want %q", got, loggedErr.Error())
	}
	if _, ok := events[0].Tags["error"]; ok {
		t.Error("例外へ変換した error 属性をタグにも残してはならない")
	}
	if got := events[0].Tags["request_id"]; got != "req-123" {
		t.Errorf("Sentry イベントの request_id タグが期待と異なる: got %q, want %q", got, "req-123")
	}
	if capturedHint == nil {
		t.Fatal("BeforeSend に EventHint が渡されていない")
	}
	if !errors.Is(capturedHint.OriginalException, loggedErr) {
		t.Errorf("EventHint の OriginalException が期待と異なる: got %v, want %v", capturedHint.OriginalException, loggedErr)
	}
	if capturedHint.Context != ctx {
		t.Error("EventHint にログの context が引き継がれていない")
	}
}

func TestSlogHandler_IgnorableExceptionIsDroppedByBeforeSend(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHubWithBeforeSend(t, beforeSend)
	logger, _ := newSlogLogger()
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	logger.ErrorContext(ctx, "クライアント切断", "error", context.Canceled)

	if got := len(transport.Events()); got != 0 {
		t.Errorf("無視対象の例外は BeforeSend で破棄されるべき: got %d events", got)
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

	// Not sent to Sentry: only error and above are captured.
	//
	// [Ja] Sentry には送信されない (error 以上のみをキャプチャするため)。
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

	// Warn is below error, so it is not sent to Sentry. This guards the policy
	// that warning logs are operational noise and should not become issues.
	//
	// [Ja] Warn は error 未満のため Sentry には送らない。「警告ログは運用上の
	// ノイズなので Sentry に流さない」というポリシーを担保する。
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
	if got := events[0].Tags["service"]; got != "test" {
		t.Errorf("Sentry イベントの service タグが期待と異なる: got %q, want %q", got, "test")
	}
	if got := events[0].Tags["request_id"]; got != "req-123" {
		t.Errorf("Sentry イベントの request_id タグが期待と異なる: got %q, want %q", got, "req-123")
	}
}

func TestSlogHandler_ErrorAttributeFromWithAttrsIsCapturedAsException(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)
	loggedErr := errors.New("WithAttrs テスト例外")
	logger, _ := newSlogLogger()
	logger = logger.With("error", loggedErr)
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	logger.ErrorContext(ctx, "WithAttrs の例外")

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("Sentry に送信されたイベント数が期待と異なる: got %d, want 1", len(events))
	}
	if len(events[0].Exception) != 1 {
		t.Fatalf("WithAttrs の error 属性が Exception へ変換されていない: got %+v", events[0].Exception)
	}
	if got := events[0].Exception[0].Value; got != loggedErr.Error() {
		t.Errorf("Sentry Exception の値が期待と異なる: got %q, want %q", got, loggedErr.Error())
	}
}

func TestSlogHandler_WithGroupFlattensTags(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)
	logger, _ := newSlogLogger()
	logger = logger.WithGroup("http").With("method", "POST").WithGroup("request")
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	logger.ErrorContext(
		ctx,
		"グループ属性付きエラー",
		slog.Group("user", slog.String("email", "user@example.com")),
		"path", "/posts",
	)

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("Sentry に送信されたイベント数が期待と異なる: got %d, want 1", len(events))
	}
	wantTags := map[string]string{
		"http.method":             "POST",
		"http.request.path":       "/posts",
		"http.request.user.email": "user@example.com",
	}
	for key, want := range wantTags {
		if got := events[0].Tags[key]; got != want {
			t.Errorf("Sentry イベントの %s タグが期待と異なる: got %q, want %q", key, got, want)
		}
	}
}

func TestSentryLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level slog.Level
		want  sentry.Level
	}{
		{name: "Error", level: slog.LevelError, want: sentry.LevelError},
		{name: "Fatal", level: levelFatal, want: sentry.LevelFatal},
		{name: "Fatalより高いレベル", level: levelFatal + 4, want: sentry.LevelFatal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sentryLevel(tt.level); got != tt.want {
				t.Errorf("sentryLevel() = %q, want %q", got, tt.want)
			}
		})
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
