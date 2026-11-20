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

// slogFakeTransportはsentry.Transportの実装で、送信されたイベントをすべて記録する。
// slog_handler_test専用に閉じておくため、既存のsentry_test.goの構造体名と被らない名前にする。
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

// テストごとに独立したHubとfake transportを返す。
func newSlogTestHub(t *testing.T) (*sentry.Hub, *slogFakeTransport) {
	t.Helper()
	return newSlogTestHubWithBeforeSend(t, nil)
}

// 指定されたBeforeSend callbackを設定した独立テスト用Hubを返す。
// グローバルなsentry.Initは呼ばないため、並行テスト間でSentryのグローバル状態を
// 介した干渉は起きない。
func newSlogTestHubWithBeforeSend(
	t *testing.T,
	beforeSend func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event,
) (*sentry.Hub, *slogFakeTransport) {
	t.Helper()

	transport := &slogFakeTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		// 有効なDSNを渡さないとclientがイベントを処理しないため、ダミーDSNを設定する
		Dsn:         "https://public@example.com/1",
		Transport:   transport,
		Environment: "test",
		BeforeSend:  beforeSend,
	})
	if err != nil {
		t.Fatalf("sentry.NewClient()のエラー = %v", err)
	}
	return sentry.NewHub(client, sentry.NewScope()), transport
}

// newSlogLoggerはbaseハンドラー (バッファ書き込み) とSentryハンドラーを合成したloggerを返す。
// バッファをアサーションで確認することで、baseハンドラーに「常に」ログが届くことを検証できる。
func newSlogLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(NewSlogHandler(base)), &buf
}

func TestSlogHandler_ErrorIsCapturedToSentry(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)
	logger, buf := newSlogLogger()

	// contextにテスト用Hubを載せると、SentryハンドラーがそのHubに
	// eventをキャプチャする。
	ctx := sentry.SetHubOnContext(context.Background(), hub)
	logger.ErrorContext(ctx, "テストエラー", "key", "value")

	// baseハンドラーには通常通りログが書き込まれる
	if !strings.Contains(buf.String(), "テストエラー") {
		t.Errorf("baseハンドラーにエラーログが書かれていない: 実測値 = %q", buf.String())
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("Sentryに送信されたイベント数が期待と異なる: 実測値 = %d、期待値 = 1", len(events))
	}
	if events[0].Level != sentry.LevelError {
		t.Errorf("SentryイベントのLevelが期待と異なる: 実測値 = %q、期待値 = %q", events[0].Level, sentry.LevelError)
	}
	if events[0].Message != "テストエラー" {
		t.Errorf("SentryイベントのMessageが期待と異なる: 実測値 = %q、期待値 = %q", events[0].Message, "テストエラー")
	}
	if events[0].Logger != "slog" {
		t.Errorf("SentryイベントのLoggerが期待と異なる: 実測値 = %q、期待値 = %q", events[0].Logger, "slog")
	}
	if got := events[0].Tags["key"]; got != "value" {
		t.Errorf("Sentryイベントのkeyタグが期待と異なる: 実測値 = %q、期待値 = %q", got, "value")
	}
	if len(events[0].Exception) != 0 {
		t.Errorf("error属性がないログにはExceptionがないはず: 実測値 = %+v", events[0].Exception)
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
		t.Fatalf("Sentryに送信されたイベント数が期待と異なる: 実測値 = %d、期待値 = 1", len(events))
	}
	if len(events[0].Exception) != 1 {
		t.Fatalf("SentryイベントのException数が期待と異なる: 実測値 = %d、期待値 = 1", len(events[0].Exception))
	}
	if got := events[0].Exception[0].Value; got != loggedErr.Error() {
		t.Errorf("Sentry Exceptionの値が期待と異なる: 実測値 = %q、期待値 = %q", got, loggedErr.Error())
	}
	if _, ok := events[0].Tags["error"]; ok {
		t.Error("例外へ変換したerror属性をタグにも残してはならない")
	}
	if got := events[0].Tags["request_id"]; got != "req-123" {
		t.Errorf("Sentryイベントのrequest_idタグが期待と異なる: 実測値 = %q、期待値 = %q", got, "req-123")
	}
	if capturedHint == nil {
		t.Fatal("BeforeSendにEventHintが渡されていない")
	}
	if !errors.Is(capturedHint.OriginalException, loggedErr) {
		t.Errorf("EventHintのOriginalExceptionが期待と異なる: 実測値 = %v、期待値 = %v", capturedHint.OriginalException, loggedErr)
	}
	if capturedHint.Context != ctx {
		t.Error("EventHintにログのcontextが引き継がれていない")
	}
}

func TestSlogHandler_IgnorableExceptionIsDroppedByBeforeSend(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHubWithBeforeSend(t, beforeSend)
	logger, _ := newSlogLogger()
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	logger.ErrorContext(ctx, "クライアント切断", "error", context.Canceled)

	if got := len(transport.Events()); got != 0 {
		t.Errorf("無視対象の例外はBeforeSendで破棄されるべき: 実測値 = %d件のイベント", got)
	}
}

func TestSlogHandler_InfoIsNotCapturedToSentry(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)
	logger, buf := newSlogLogger()

	ctx := sentry.SetHubOnContext(context.Background(), hub)
	logger.InfoContext(ctx, "テスト情報", "key", "value")

	// baseハンドラーにはInfoログが書き込まれる
	if !strings.Contains(buf.String(), "テスト情報") {
		t.Errorf("baseハンドラーにInfoログが書かれていない: 実測値 = %q", buf.String())
	}

	// Sentryには送信されない (error以上のみをキャプチャするため)。
	events := transport.Events()
	if len(events) != 0 {
		t.Errorf("InfoレベルではSentryに送信されないはず: 実測値 = %d件のイベント", len(events))
	}
}

func TestSlogHandler_WarnIsNotCapturedToSentry(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)
	logger, buf := newSlogLogger()

	ctx := sentry.SetHubOnContext(context.Background(), hub)
	logger.WarnContext(ctx, "テスト警告")

	if !strings.Contains(buf.String(), "テスト警告") {
		t.Errorf("baseハンドラーにWarnログが書かれていない: 実測値 = %q", buf.String())
	}

	// Warnはerror未満のためSentryには送らない。「警告ログは運用上の
	// ノイズなのでSentryに流さない」というポリシーを担保する。
	events := transport.Events()
	if len(events) != 0 {
		t.Errorf("WarnレベルではSentryに送信されないはず: 実測値 = %d件のイベント", len(events))
	}
}

func TestSlogHandler_WithAttrs_PropagatesToBothHandlers(t *testing.T) {
	t.Parallel()

	hub, transport := newSlogTestHub(t)

	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	// `slog.With(...)` は内部でHandler.WithAttrsを呼ぶ。これがmultiHandler経由で
	// base / sentryの両方に伝播することを担保する。
	logger := slog.New(NewSlogHandler(base)).With("service", "test", "request_id", "req-123")

	ctx := sentry.SetHubOnContext(context.Background(), hub)
	logger.ErrorContext(ctx, "属性付きエラー")

	// baseハンドラーにはWithAttrsで設定した属性が反映される
	output := buf.String()
	if !strings.Contains(output, `service=test`) {
		t.Errorf("baseハンドラーにservice属性が反映されていない: 実測値 = %q", output)
	}
	if !strings.Contains(output, `request_id=req-123`) {
		t.Errorf("baseハンドラーにrequest_id属性が反映されていない: 実測値 = %q", output)
	}

	// Sentryにもイベントが届く
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("Sentryに送信されたイベント数が期待と異なる: 実測値 = %d、期待値 = 1", len(events))
	}
	if events[0].Message != "属性付きエラー" {
		t.Errorf("SentryイベントのMessageが期待と異なる: 実測値 = %q、期待値 = %q", events[0].Message, "属性付きエラー")
	}
	if got := events[0].Tags["service"]; got != "test" {
		t.Errorf("Sentryイベントのserviceタグが期待と異なる: 実測値 = %q、期待値 = %q", got, "test")
	}
	if got := events[0].Tags["request_id"]; got != "req-123" {
		t.Errorf("Sentryイベントのrequest_idタグが期待と異なる: 実測値 = %q、期待値 = %q", got, "req-123")
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
		t.Fatalf("Sentryに送信されたイベント数が期待と異なる: 実測値 = %d、期待値 = 1", len(events))
	}
	if len(events[0].Exception) != 1 {
		t.Fatalf("WithAttrsのerror属性がExceptionへ変換されていない: 実測値 = %+v", events[0].Exception)
	}
	if got := events[0].Exception[0].Value; got != loggedErr.Error() {
		t.Errorf("Sentry Exceptionの値が期待と異なる: 実測値 = %q、期待値 = %q", got, loggedErr.Error())
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
		t.Fatalf("Sentryに送信されたイベント数が期待と異なる: 実測値 = %d、期待値 = 1", len(events))
	}
	wantTags := map[string]string{
		"http.method":             "POST",
		"http.request.path":       "/posts",
		"http.request.user.email": "user@example.com",
	}
	for key, want := range wantTags {
		if got := events[0].Tags[key]; got != want {
			t.Errorf("Sentryイベントの%sタグが期待と異なる: 実測値 = %q、期待値 = %q", key, got, want)
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
				t.Errorf("sentryLevel() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestSlogHandler_HubFromContext_PrefersRequestHub(t *testing.T) {
	t.Parallel()

	// 2つの独立したHubを用意し、ctxにbindしたHubにだけイベントが届くことを担保する。
	// これにより「リクエストごとに別々のHubにイベントが分離される」挙動 (sentryhttp経由のリクエストHub) を再現する。
	hubA, transportA := newSlogTestHub(t)
	_, transportB := newSlogTestHub(t)

	logger, _ := newSlogLogger()

	// hubAのみをctxにbindする。
	ctx := sentry.SetHubOnContext(context.Background(), hubA)
	logger.ErrorContext(ctx, "hubA に届くべきエラー")

	if len(transportA.Events()) != 1 {
		t.Errorf("hubAに紐付いたtransportにエラーが届かなかった: 実測値 = %d件のイベント", len(transportA.Events()))
	}
	if len(transportB.Events()) != 0 {
		t.Errorf("hubBはctxにbindされていないため何も届かないはず: 実測値 = %d件のイベント", len(transportB.Events()))
	}
}
