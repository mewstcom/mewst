package sentry

import (
	"context"
	"errors"
	"log/slog"

	"github.com/getsentry/sentry-go"
)

// levelFatal is the fatal log level. slog has no built-in fatal level, so 12
// (one step above slog.LevelError) is used and mapped onto sentry.LevelFatal.
//
// [Ja] levelFatal は fatal ログレベル。slog には fatal レベルの組み込みが
// ないため、slog.LevelError の 1 段上の 12 を使い sentry.LevelFatal に対応させる。
const levelFatal = slog.Level(12)

// NewSlogHandler returns a slog.Handler that fans out to the base handler and a
// Sentry handler.
//
// Every log record reaches the base handler (normal stdout logging), while
// records at slog.LevelError or above are additionally captured to Sentry as
// events (= issues). This lets slog.ErrorContext calls made in any layer
// (Handler / UseCase / Validator / Repository / Middleware) reach Sentry
// automatically, without each layer calling sentry.CaptureError explicitly.
//
// When Sentry is uninitialized (empty DSN), the Sentry handler resolves to the
// no-op CurrentHub, so composing it is always safe.
//
// [Ja] NewSlogHandler は base ハンドラーと Sentry ハンドラーに fan-out する
// slog.Handler を返す。
//
// すべてのログは base ハンドラーに届く (通常の標準出力) 一方、slog.LevelError
// 以上のログは加えて Sentry の event (= issue) として送られる。これにより各層
// (Handler / UseCase / Validator / Repository / Middleware) の slog.ErrorContext
// が自動的に Sentry に届き、各層で sentry.CaptureError を明示的に呼ぶ必要がない。
//
// Sentry が未初期化 (DSN 空) の場合、Sentry ハンドラーは no-op の CurrentHub に
// 解決されるため、合成は常に安全。
func NewSlogHandler(base slog.Handler) slog.Handler {
	return &multiHandler{handlers: []slog.Handler{base, &sentryHandler{}}}
}

// eventAttribute is a flattened slog attribute ready to be attached to a
// Sentry event. A top-level "error" or "err" attribute also keeps the
// original error so the handler can preserve exception details.
//
// [Ja] eventAttribute は Sentry event へ付与できる形に平坦化した slog 属性。
// トップレベルの "error" / "err" 属性では、例外情報を維持できるよう元の error
// も保持する。
type eventAttribute struct {
	key       string
	value     string
	exception error
}

// sentryHandler is a slog.Handler that captures records at slog.LevelError and
// above as Sentry events (issues) by calling hub.CaptureEventWithHint. Top-level
// error attributes are converted to Event.Exception and passed as the hint's
// OriginalException.
//
// The hub is taken from the record's context when present, so events stay bound
// to the per-request or per-job hub (e.g. one set by sentryhttp or the River
// middleware); otherwise the current hub is used.
//
// slog attributes are attached as event tags so that the sensitive-tag masking
// in beforeSend (see filterTags) keeps applying to PII carried as attributes.
//
// [Ja] sentryHandler は slog.LevelError 以上のログを hub.CaptureEventWithHint で
// Sentry の event (= issue) として送る slog.Handler。トップレベルの error 属性は
// Event.Exception へ変換し、hint の OriginalException としても渡す。
//
// Hub は record の context に載っていればそれを使うため、event はリクエスト単位・
// ジョブ単位の Hub (例: sentryhttp や River ミドルウェアが設定した Hub) に紐付く。
// 無ければ current hub を使う。
//
// slog 属性は event のタグとして付与する。これにより beforeSend の
// センシティブタグのマスキング (filterTags 参照) が、属性として運ばれる PII にも
// 引き続き適用される。
type sentryHandler struct {
	// attrs holds attributes accumulated via WithAttrs, already flattened and
	// prefixed with the group path that was active when they were added.
	//
	// [Ja] attrs は WithAttrs で蓄積した属性。追加時点で有効だったグループパスの
	// プレフィックスを付けて平坦化済み。
	attrs []eventAttribute

	// groupPrefix is the dotted group path applied to record attributes at
	// Handle time (e.g. "http.request.").
	//
	// [Ja] groupPrefix は Handle 時に record の属性へ適用するドット区切りの
	// グループパス (例: "http.request.")。
	groupPrefix string
}

// Enabled reports whether the level is captured to Sentry. Only slog.LevelError
// and above are, so info/warn logs stay out of Sentry.
//
// [Ja] Enabled はそのレベルを Sentry に送るかを返す。slog.LevelError 以上のみ
// true とし、info / warn ログは Sentry に流さない。
func (h *sentryHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelError
}

// Handle builds a Sentry event from the record and captures it on the hub
// resolved from ctx (falling back to the current hub).
//
// [Ja] Handle は record から Sentry event を組み立て、ctx から解決した Hub
// (無ければ current hub) でキャプチャする。
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

// WithAttrs returns a handler that also carries attrs, flattened under the
// current group prefix.
//
// [Ja] WithAttrs は attrs を現在のグループプレフィックス配下に平坦化して
// 併せ持つハンドラーを返す。
func (h *sentryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	eventAttrs := make([]eventAttribute, len(h.attrs), len(h.attrs)+len(attrs))
	copy(eventAttrs, h.attrs)
	for _, attr := range attrs {
		eventAttrs = append(eventAttrs, flattenAttr(h.groupPrefix, attr)...)
	}
	return &sentryHandler{attrs: eventAttrs, groupPrefix: h.groupPrefix}
}

// WithGroup returns a handler that nests subsequent attributes under name.
//
// [Ja] WithGroup は以降の属性を name 配下にネストするハンドラーを返す。
func (h *sentryHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &sentryHandler{attrs: h.attrs, groupPrefix: h.groupPrefix + name + "."}
}

// sentryLevel maps a slog level to a Sentry level. Only error and above reach
// this handler, so the mapping distinguishes error from fatal.
//
// [Ja] sentryLevel は slog レベルを Sentry レベルに対応させる。本ハンドラーには
// error 以上しか来ないため、error と fatal を区別する。
func sentryLevel(level slog.Level) sentry.Level {
	if level >= levelFatal {
		return sentry.LevelFatal
	}
	return sentry.LevelError
}

// flattenAttr flattens a slog attribute into tags, expanding group attributes
// into dotted keys (e.g. "user.email") so that filterTags can match sensitive
// leaf keys.
//
// [Ja] flattenAttr は slog 属性を tag に平坦化する。グループ属性はドット区切りの
// キー (例: "user.email") に展開し、filterTags がセンシティブな末端キーに
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

// multiHandler fans log records out to multiple slog handlers. Keeping the
// fan-out implementation local avoids an additional dependency for this small
// composition primitive.
//
// [Ja] multiHandler は複数の slog.Handler にログを fan-out する。この小さな合成
// 処理をローカルに実装し、追加の外部依存を避ける。
type multiHandler struct {
	handlers []slog.Handler
}

// Enabled reports whether any child handler accepts the level.
//
// [Ja] 子ハンドラーのいずれかが指定レベルを受け付ける場合に true を返す。
func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle fans a cloned record out to each enabled child handler to avoid
// aliasing the record's attribute storage.
//
// [Ja] 有効な各子ハンドラーへ record の独立したコピーを fan-out し、属性格納領域の
// alias による競合を避ける。
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

// WithAttrs propagates attributes to every child handler.
//
// [Ja] すべての子ハンドラーへ属性を伝播させる。
func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		out[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: out}
}

// WithGroup propagates a group name to every child handler.
//
// [Ja] すべての子ハンドラーへグループ名を伝播させる。
func (h *multiHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		out[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: out}
}
