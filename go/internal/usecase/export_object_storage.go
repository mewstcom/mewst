package usecase

import (
	"context"
	"io"
	"time"
)

// ExportObjectStorage is the Application-layer port for the object storage
// that holds export zip archives. The S3 (Cloudflare R2) implementation lives
// in internal/storage; use cases depend only on this interface so that tests
// can substitute a fake without touching the AWS SDK.
//
// [Ja] ExportObjectStorage はエクスポート zip を保持するオブジェクトストレージの
// Application 層 port。S3 (Cloudflare R2) 実装は internal/storage に置き、
// UseCase はこの interface にだけ依存することで、テストでは AWS SDK に触れずに
// fake へ差し替えられる。
type ExportObjectStorage interface {
	// Upload streams body to the object storage under key. Implementations
	// must upload in bounded chunks and must not buffer the whole body in
	// memory, so that archive generation stays O(1) in memory regardless of
	// the archive size.
	//
	// [Ja] Upload は body を key の位置へストリーミングアップロードする。
	// 実装は一定サイズのチャンク単位でアップロードし、body 全体をメモリに
	// 保持してはならない (アーカイブサイズに依らず生成処理のメモリを O(1) に保つ)。
	Upload(ctx context.Context, key string, body io.Reader) error

	// Download returns the object body as a stream together with its size in
	// bytes. The caller must close the returned io.ReadCloser.
	//
	// [Ja] Download はオブジェクト本体のストリームとバイト単位のサイズを返す。
	// 返された io.ReadCloser は呼び出し側が必ず閉じる。
	Download(ctx context.Context, key string) (io.ReadCloser, int64, error)

	// Delete removes the object stored under key. Deleting a key that no
	// longer exists is treated as success so that retried cleanup jobs stay
	// idempotent.
	//
	// [Ja] Delete は key の位置のオブジェクトを削除する。既に存在しない key の
	// 削除は成功として扱い、リトライされる cleanup ジョブを冪等に保つ。
	Delete(ctx context.Context, key string) error

	// ListPrefix visits every object whose key starts with prefix and sorts
	// strictly after startAfter, passing its key and last-modified time to
	// yield. An empty startAfter begins at the first object under the prefix.
	// Implementations must follow the storage's pagination until the listing is
	// exhausted so callers never receive a silently truncated result. An error
	// returned by yield stops the listing and is returned to the caller.
	//
	// startAfter lets a caller whose walk is bounded resume where the previous
	// one stopped, so a prefix too large for one walk is covered by several
	// instead of restarting from the beginning every time.
	//
	// [Ja] ListPrefix は key が prefix で始まり startAfter より厳密に後ろへ並ぶ全
	// オブジェクトを走査し、key と最終更新時刻を yield に渡す。startAfter が空の
	// 場合は prefix 配下の先頭から始める。実装はストレージのページングを最後まで辿り、
	// 呼び出し側が黙って切り詰められた結果を受け取ることがないようにする。
	// yield がエラーを返した場合は走査を止め、そのエラーを呼び出し側へ返す。
	//
	// startAfter により、走査が有界な呼び出し側は前回止まった位置から再開できる。
	// 1 回の走査に収まらないプレフィックスを、毎回先頭からやり直すのではなく複数回で
	// 網羅するため。
	ListPrefix(ctx context.Context, prefix, startAfter string, yield func(key string, lastModified time.Time) error) error
}
