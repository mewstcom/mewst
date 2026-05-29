package sentry

import (
	"context"
	"errors"
	"log/slog"

	sentryslog "github.com/getsentry/sentry-go/slog"
)

// NewSlogHandler は標準出力ハンドラーと Sentry ハンドラーを合成した slog.Handler を返す。
//
// 合成後のハンドラーを `slog.SetDefault(slog.New(NewSlogHandler(base)))` のように設定すると、
// 以下の挙動になる:
//   - すべてのレベルのログは引数の base ハンドラーに転送される (通常の標準出力)
//   - `slog.LevelError` 以上のログは Sentry のイベント (= イシュー) として送信される
//
// この設計により、Handler / UseCase / Validator / Repository / Middleware など各層で呼ぶ
// `slog.ErrorContext(ctx, ...)` が自動的に Sentry に届く。各層で `sentry.CaptureError` を
// 明示的に呼ぶ必要がない。
//
// Sentry が未初期化 (DSN 空) の場合でも、Sentry ハンドラー側は no-op で動作するため
// 安全に合成できる。
//
// 制限: sentry-go v0.46.2 の `sentryslog.Option.EventLevel` は将来のバージョン (v0.48.0)
// で削除される予定。削除されたタイミングで `sentry.CaptureException` 直接呼び出しベースの
// 設計へ移行する必要がある (Logs API を採用するか slog.Handler 内で CaptureException を
// 呼ぶ独自実装に切り替えるかを判断する)。
func NewSlogHandler(base slog.Handler) slog.Handler {
	sentryHandler := sentryslog.Option{
		// slog.LevelError 以上を Sentry のイベントとして送信する。
		// sentryslog.LevelFatal も含めることで、将来 Fatal レベルを使った場合も自動的に拾う。
		// EventLevel は v0.48.0 で削除される予定だが、自動キャプチャを実現するため明示的に使う。
		EventLevel: []slog.Level{slog.LevelError, sentryslog.LevelFatal}, //nolint:staticcheck // 自動キャプチャ実現のため意図的に使用
		// Sentry Logs API (構造化ログ) は使用しない。
		// 空スライスを渡すことで logHandler を完全に無効化する (nil だと全レベルが Logs API に流れる)。
		LogLevel: []slog.Level{},
	}.NewSentryHandler(context.Background())

	return &multiHandler{handlers: []slog.Handler{base, sentryHandler}}
}

// multiHandler は複数の slog.Handler に対してログをファンアウトする。
//
// `samber/slog-multi` 等の外部依存に頼らず、ファンアウトの仕組みを内製化することで、
// 標準ライブラリと sentryslog のみで完結させる。
type multiHandler struct {
	handlers []slog.Handler
}

// Enabled はいずれかのサブハンドラーが指定レベルを受け付けるなら true を返す。
func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle は Record を各サブハンドラーにファンアウトする。
// 各サブハンドラーには Record の独立コピーを渡し、属性スライスのエイリアスによる競合を避ける。
func (h *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var errs []error
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := handler.Handle(ctx, record.Clone()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// WithAttrs はサブハンドラーすべてに属性を伝播させる。
func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		out[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: out}
}

// WithGroup はサブハンドラーすべてにグループ名を伝播させる。
func (h *multiHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		out[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: out}
}
