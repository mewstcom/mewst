package sentry

import (
	"context"
	"errors"
	"log/slog"

	"github.com/getsentry/sentry-go"
)

// levelFatalはfatalログレベル。slogにはfatalレベルの組み込みが
// ないため、slog.LevelErrorの1段上の12を使いsentry.LevelFatalに対応させる。
const levelFatal = slog.Level(12)

// NewSlogHandlerはbaseハンドラーとSentryハンドラーにfan-outする
// slog.Handlerを返す。
//
// すべてのログはbaseハンドラーに届く (通常の標準出力) 一方、slog.LevelError
// 以上のログは加えてSentryのevent (= issue) として送られる。これにより各層
// (Handler / UseCase / Validator / Repository / Middleware) のslog.ErrorContext
// が自動的にSentryに届き、各層でsentry.CaptureErrorを明示的に呼ぶ必要がない。
//
// Sentryが未初期化 (DSN空) の場合、Sentryハンドラーはno-opのCurrentHubに
// 解決されるため、合成は常に安全。
func NewSlogHandler(base slog.Handler) slog.Handler {
	return &multiHandler{handlers: []slog.Handler{base, &sentryHandler{}}}
}

// eventAttributeはSentry eventへ付与できる形に平坦化したslog属性。
// トップレベルの "error" / "err" 属性では、例外情報を維持できるよう元のerror
// も保持する。
type eventAttribute struct {
	key       string
	value     string
	exception error
}

// sentryHandlerはslog.LevelError以上のログをhub.CaptureEventWithHintで
// Sentryのevent (= issue) として送るslog.Handler。トップレベルのerror属性は
// Event.Exceptionへ変換し、hintのOriginalExceptionとしても渡す。
//
// Hubはrecordのcontextに載っていればそれを使うため、eventはリクエスト単位・
// ジョブ単位のHub (例: sentryhttpやRiverミドルウェアが設定したHub) に紐付く。
// 無ければcurrent hubを使う。
//
// slog属性はeventのタグとして付与する。これによりbeforeSendの
// センシティブタグのマスキング (filterTags参照) が、属性として運ばれるPIIにも
// 引き続き適用される。
type sentryHandler struct {
	// attrsはWithAttrsで蓄積した属性。追加時点で有効だったグループパスの
	// プレフィックスを付けて平坦化済み。
	attrs []eventAttribute

	// groupPrefixはHandle時にrecordの属性へ適用するドット区切りの
	// グループパス (例: "http.request.")。
	groupPrefix string
}

// EnabledはそのレベルをSentryに送るかを返す。slog.LevelError以上のみ
// trueとし、info / warnログはSentryに流さない。
func (h *sentryHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelError
}

// HandleはrecordからSentry eventを組み立て、ctxから解決したHub
// (無ければcurrent hub) でキャプチャする。
func (h *sentryHandler) Handle(ctx context.Context, record slog.Record) error {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	event := sentry.NewEvent()
	event.Level = sentryLevel(record.Level)
	event.Logger = "slog"
	event.Message = record.Message
	if !record.Time.IsZero() {
		event.Timestamp = record.Time
	}

	var originalException error
	addAttrs := func(attrs []eventAttribute) {
		for _, attr := range attrs {
			if originalException == nil && attr.exception != nil {
				originalException = attr.exception
				continue
			}
			event.Tags[attr.key] = attr.value
		}
	}
	addAttrs(h.attrs)
	record.Attrs(func(attr slog.Attr) bool {
		addAttrs(flattenAttr(h.groupPrefix, attr))
		return true
	})

	if originalException != nil {
		if client := hub.Client(); client != nil {
			event.SetException(originalException, client.Options().MaxErrorDepth)
		}
	}
	hub.CaptureEventWithHint(event, &sentry.EventHint{
		Context:           ctx,
		OriginalException: originalException,
	})
	return nil
}

// WithAttrsはattrsを現在のグループプレフィックス配下に平坦化して
// 併せ持つハンドラーを返す。
func (h *sentryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	eventAttrs := make([]eventAttribute, len(h.attrs), len(h.attrs)+len(attrs))
	copy(eventAttrs, h.attrs)
	for _, attr := range attrs {
		eventAttrs = append(eventAttrs, flattenAttr(h.groupPrefix, attr)...)
	}
	return &sentryHandler{attrs: eventAttrs, groupPrefix: h.groupPrefix}
}

// WithGroupは以降の属性をname配下にネストするハンドラーを返す。
func (h *sentryHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &sentryHandler{attrs: h.attrs, groupPrefix: h.groupPrefix + name + "."}
}

// sentryLevelはslogレベルをSentryレベルに対応させる。本ハンドラーには
// error以上しか来ないため、errorとfatalを区別する。
func sentryLevel(level slog.Level) sentry.Level {
	if level >= levelFatal {
		return sentry.LevelFatal
	}
	return sentry.LevelError
}

// flattenAttrはslog属性をtagに平坦化する。グループ属性はドット区切りの
// キー (例: "user.email") に展開し、filterTagsがセンシティブな末端キーに
// マッチできるようにする。
func flattenAttr(prefix string, attr slog.Attr) []eventAttribute {
	value := attr.Value.Resolve()
	if value.Kind() == slog.KindGroup {
		groupPrefix := prefix
		if attr.Key != "" {
			groupPrefix = prefix + attr.Key + "."
		}
		var attrs []eventAttribute
		for _, groupAttr := range value.Group() {
			attrs = append(attrs, flattenAttr(groupPrefix, groupAttr)...)
		}
		return attrs
	}
	if attr.Key == "" {
		return nil
	}

	var exception error
	if prefix == "" && (attr.Key == "error" || attr.Key == "err") {
		exception, _ = value.Any().(error)
	}
	return []eventAttribute{{
		key:       prefix + attr.Key,
		value:     value.String(),
		exception: exception,
	}}
}

// multiHandlerは複数のslog.Handlerにログをfan-outする。この小さな合成
// 処理をローカルに実装し、追加の外部依存を避ける。
type multiHandler struct {
	handlers []slog.Handler
}

// 子ハンドラーのいずれかが指定レベルを受け付ける場合にtrueを返す。
func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// 有効な各子ハンドラーへrecordの独立したコピーをfan-outし、属性格納領域の
// aliasによる競合を避ける。
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

// すべての子ハンドラーへ属性を伝播させる。
func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		out[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: out}
}

// すべての子ハンドラーへグループ名を伝播させる。
func (h *multiHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		out[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: out}
}
