package usecase

import (
	"context"
	"io"
	"time"
)

// ExportObjectStorageはエクスポートzipを保持するオブジェクトストレージの
// Application層port。S3 (Cloudflare R2) 実装はinternal/storageに置き、
// UseCaseはこのinterfaceにだけ依存することで、テストではAWS SDKに触れずに
// fakeへ差し替えられる。
type ExportObjectStorage interface {
	// Uploadはbodyをkeyの位置へストリーミングアップロードする。
	// 実装は一定サイズのチャンク単位でアップロードし、body全体をメモリに
	// 保持してはならない (アーカイブサイズに依らず生成処理のメモリをO(1) に保つ)。
	Upload(ctx context.Context, key string, body io.Reader) error

	// Downloadはオブジェクト本体のストリームとバイト単位のサイズを返す。
	// 返されたio.ReadCloserは呼び出し側が必ず閉じる。
	Download(ctx context.Context, key string) (io.ReadCloser, int64, error)

	// Deleteはkeyの位置のオブジェクトを削除する。既に存在しないkeyの
	// 削除は成功として扱い、リトライされるcleanupジョブを冪等に保つ。
	Delete(ctx context.Context, key string) error

	// ListPrefixはkeyがprefixで始まりstartAfterより厳密に後ろへ並ぶ全
	// オブジェクトを走査し、keyと最終更新時刻をyieldに渡す。startAfterが空の
	// 場合はprefix配下の先頭から始める。実装はストレージのページングを最後まで辿り、
	// 呼び出し側が黙って切り詰められた結果を受け取ることがないようにする。
	// yieldがエラーを返した場合は走査を止め、そのエラーを呼び出し側へ返す。
	//
	// startAfterにより、走査が有界な呼び出し側は前回止まった位置から再開できる。
	// 1回の走査に収まらないプレフィックスを、毎回先頭からやり直すのではなく複数回で
	// 網羅するため。
	ListPrefix(ctx context.Context, prefix, startAfter string, yield func(key string, lastModified time.Time) error) error
}
