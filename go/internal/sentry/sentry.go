// Package sentryはSentryエラー追跡サービスとの連携機能を提供します
package sentry

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// ConfigはSentryの設定を保持する
type Config struct {
	DSN              string
	Environment      string
	Release          string
	TracesSampleRate float64
	Debug            bool
}

const maskedValue = "[FILTERED]"

// sensitiveHeadersはマスクすべきHTTPヘッダー名のリスト (小文字、完全一致)
var sensitiveHeaders = []string{
	"authorization",
	"cookie",
	"x-csrf-token",
}

// sensitiveBodyKeysはマスクすべきリクエストボディのキー (部分一致、小文字)
var sensitiveBodyKeys = []string{
	"password",
	"token",
	"secret",
}

// sensitiveQueryKeysはマスクすべきクエリパラメータのキー (部分一致、小文字)
var sensitiveQueryKeys = []string{
	"token",
	"key",
}

// マスクすべきタグのキー (部分一致、小文字)。独自のSentry slogハンドラーが
// slog属性をSentryのタグへ載せるため、構造化属性としてログに載せたPII (例:
// メール送信失敗ログの "email") は、このフィルタがないとマスクされずにSentryへ
// 届く。標準エラー出力側のログには元の値が残るため、デバッグはそちらで行える。
var sensitiveTagKeys = []string{
	"email",
	"password",
	"secret",
	"token",
}

// ignoredErrorPatternsはメッセージレベルでSentry送信をスキップするパターン (正規表現)。
// クライアント切断由来のノイズやGo runtimeの正常な中断をフィルタする。
var ignoredErrorPatterns = []string{
	"context canceled",
	"net/http: abort Handler",
}

// InitはSentryを初期化する
// DSNが空の場合は初期化をスキップしnilを返す (開発環境でSentryを使用しない場合)
func Init(cfg Config) error {
	if cfg.DSN == "" {
		slog.Info("Sentry DSNが設定されていないため、Sentryは無効化されています")
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		TracesSampleRate: cfg.TracesSampleRate,
		EnableTracing:    true,
		Debug:            cfg.Debug,
		BeforeSend:       beforeSend,
		IgnoreErrors:     ignoredErrorPatterns,
	})
	if err != nil {
		return err
	}

	slog.Info("Sentryを初期化しました",
		"environment", cfg.Environment,
		"release", cfg.Release,
		"traces_sample_rate", cfg.TracesSampleRate,
	)
	return nil
}

// beforeSendはSentryにイベントを送信する前にフィルタリングを行う。
// クライアント切断や正常な中断由来のエラーは破棄し、それ以外はセンシティブデータをマスクする。
func beforeSend(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	if hint != nil && shouldDropError(hint.OriginalException) {
		return nil
	}

	filterTags(event)

	if event.Request != nil {
		filterRequestHeaders(event.Request)
		filterRequestData(event.Request)
		filterQueryString(event.Request)
	}
	return event
}

// shouldDropErrorはクライアント切断・runtime中断由来のエラーかを判定する。
// 該当する場合はSentryに送らない。
func shouldDropError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	if errors.Is(err, http.ErrAbortHandler) {
		return true
	}
	return false
}

// センシティブなタグ (メールアドレス等) をマスクする。下のリクエスト系
// フィルタと異なり、独自のSentry slogハンドラーがslog属性をevent.Tagsへ
// 格納して生成したイベントも対象にする。
func filterTags(event *sentry.Event) {
	for key := range event.Tags {
		lowerKey := strings.ToLower(key)
		for _, sensitive := range sensitiveTagKeys {
			if strings.Contains(lowerKey, sensitive) {
				event.Tags[key] = maskedValue
				break
			}
		}
	}
}

// filterRequestHeadersはセンシティブなHTTPヘッダーをマスクする
func filterRequestHeaders(req *sentry.Request) {
	if req.Headers == nil {
		return
	}

	for headerName := range req.Headers {
		lowerName := strings.ToLower(headerName)
		for _, sensitive := range sensitiveHeaders {
			if lowerName == sensitive {
				req.Headers[headerName] = maskedValue
				break
			}
		}
	}
}

// filterRequestDataはセンシティブなリクエストボディのフィールドをマスクする
func filterRequestData(req *sentry.Request) {
	if req.Data == "" {
		return
	}

	values, err := url.ParseQuery(req.Data)
	if err != nil {
		// パースできない場合は安全のためデータ全体を削除
		req.Data = maskedValue
		return
	}

	filtered := false
	for key := range values {
		lowerKey := strings.ToLower(key)
		for _, sensitive := range sensitiveBodyKeys {
			if strings.Contains(lowerKey, sensitive) {
				values.Set(key, maskedValue)
				filtered = true
				break
			}
		}
	}

	if filtered {
		req.Data = values.Encode()
	}
}

// filterQueryStringはセンシティブなクエリパラメータをマスクする
func filterQueryString(req *sentry.Request) {
	if req.QueryString == "" {
		return
	}

	values, err := url.ParseQuery(req.QueryString)
	if err != nil {
		// パースできない場合は安全のためクエリ全体を削除
		req.QueryString = maskedValue
		return
	}

	filtered := false
	for key := range values {
		lowerKey := strings.ToLower(key)
		for _, sensitive := range sensitiveQueryKeys {
			if strings.Contains(lowerKey, sensitive) {
				values.Set(key, maskedValue)
				filtered = true
				break
			}
		}
	}

	if filtered {
		req.QueryString = values.Encode()
	}
}

// FlushはバッファリングされたイベントをSentryに送信する。
// アプリケーション終了時に呼び出す。
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// CaptureErrorはエラーをSentryに送信する
func CaptureError(ctx context.Context, err error) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureException(err)
	} else {
		sentry.CaptureException(err)
	}
}

// CaptureMessageはメッセージをSentryに送信する
func CaptureMessage(ctx context.Context, message string) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureMessage(message)
	} else {
		sentry.CaptureMessage(message)
	}
}
