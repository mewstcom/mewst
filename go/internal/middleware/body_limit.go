package middleware

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// DefaultMaxBodyBytes is the default request body size limit (1 MiB).
// It generously covers the known text input forms (posts, account forms, etc.)
// while preventing memory exhaustion from intentionally large payloads.
//
// [Ja] DefaultMaxBodyBytes はリクエストボディのデフォルトサイズ上限 (1 MiB)。
// 既知のテキスト入力フォーム (ポスト・アカウントフォームなど) を余裕を持ってカバーしつつ、
// 意図的な大容量送信によるメモリ枯渇を防ぐために設定している。
const DefaultMaxBodyBytes = 1 << 20

// BodyLimit is a middleware that limits the request body size to [DefaultMaxBodyBytes].
// Requests exceeding the limit receive 413 Payload Too Large.
//
// It must be placed before any handler / middleware that calls r.ParseForm /
// r.FormValue / r.PostFormValue (these read the body as soon as they are called,
// so the body has to be wrapped beforehand).
//
// The standard [http.MaxBytesHandler] only wraps r.Body and does not respond
// with 413 automatically, so this middleware pre-reads the body, detects the
// error, and returns 413 explicitly. The pre-read body is handed to downstream
// handlers wrapped in [io.NopCloser], so the existing r.FormValue / r.ParseForm
// behavior is preserved.
//
// [Ja] BodyLimit はリクエストボディのサイズを [DefaultMaxBodyBytes] に制限するミドルウェア。
// 上限を超過したリクエストには 413 Payload Too Large を返す。
//
// r.ParseForm / r.FormValue / r.PostFormValue を呼ぶハンドラー・ミドルウェアより前に
// 配置する必要がある (これらは呼び出しと同時にボディを読み込むため、事前にラップしておく)。
//
// net/http 標準の [http.MaxBytesHandler] は r.Body をラップするだけで自動 413 応答は
// 行わないため、ここではミドルウェア側でボディを先読みしてエラーを検出し、413 を明示的に返す。
// 下流ハンドラーには先読み済みのボディを [io.NopCloser] でラップして渡すため、既存の
// r.FormValue / r.ParseForm の挙動は維持される。
func BodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pass through requests without a body (GET/HEAD/OPTIONS etc.).
		// [Ja] ボディがないリクエスト (GET/HEAD/OPTIONS 等) はそのまま通す
		if r.Body == nil || r.Body == http.NoBody {
			next.ServeHTTP(w, r)
			return
		}

		// When Content-Length already tells us the body exceeds the limit, return 413
		// without reading the body. With chunked transfer etc. ContentLength is -1,
		// so we do not reject here and leave the decision to MaxBytesReader below.
		//
		// [Ja] Content-Length が上限を超えていることが事前に分かる場合は、ボディを読まずに
		// 413 を返す。chunked 転送等で ContentLength が不明な場合は -1 になるため、
		// ここでは拒否せず後段の MaxBytesReader に判定を委ねる。
		if r.ContentLength > DefaultMaxBodyBytes {
			slog.WarnContext(r.Context(), "リクエストボディサイズ上限超過 (Content-Length)",
				"path", r.URL.Path,
				"method", r.Method,
				"content_length", r.ContentLength,
				"max", int64(DefaultMaxBodyBytes),
			)
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
			return
		}

		limited := http.MaxBytesReader(w, r.Body, DefaultMaxBodyBytes)
		buf, err := io.ReadAll(limited)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				slog.WarnContext(r.Context(), "リクエストボディサイズ上限超過 (読み込み中)",
					"path", r.URL.Path,
					"method", r.Method,
					"limit", maxBytesErr.Limit,
				)
				http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
				return
			}
			slog.ErrorContext(r.Context(), "リクエストボディの読み込み失敗",
				"error", err,
				"path", r.URL.Path,
				"method", r.Method,
			)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(buf))
		r.ContentLength = int64(len(buf))
		next.ServeHTTP(w, r)
	})
}
