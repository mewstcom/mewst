// Package storage provides Infrastructure-layer adapters for the
// S3-compatible object storage (Cloudflare R2) used by the export feature.
//
// [Ja] storage パッケージはエクスポート機能で使う S3 互換オブジェクトストレージ
// (Cloudflare R2) への Infrastructure 層アダプタを提供する。
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const (
	// exportContentType and exportCacheControl are stored as object metadata
	// so the objects carry safe defaults even when fetched outside the app.
	// The Go server still overrides the HTTP response headers on download.
	//
	// [Ja] exportContentType と exportCacheControl はオブジェクトのメタデータ
	// として保存し、アプリ外で取得された場合も安全な既定値が付くようにする。
	// ダウンロード時の HTTP レスポンスヘッダーは Go サーバーが必ず上書きする。
	exportContentType  = "application/zip"
	exportCacheControl = "private, no-store"

	// abortMultipartUploadTimeout bounds the cleanup call that aborts a
	// failed multipart upload. The cleanup context is detached from the
	// upload context, so this timeout is what keeps the abort from hanging.
	//
	// [Ja] abortMultipartUploadTimeout は失敗した multipart upload を中断する
	// cleanup 呼び出しの上限時間。cleanup context は upload の context から
	// 切り離すため、この timeout が abort のハングを防ぐ。
	abortMultipartUploadTimeout = 30 * time.Second
)

// S3Config holds the connection settings for the S3-compatible object
// storage. It mirrors the MEWST_S3_* environment variables; the caller maps
// them so this package stays independent of internal/config.
//
// [Ja] S3Config は S3 互換オブジェクトストレージへの接続設定を保持する。
// MEWST_S3_* 環境変数と対応し、呼び出し側で値を詰め替えることで本パッケージを
// internal/config から独立させる。
type S3Config struct {
	BucketName      string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

// S3ExportStorage is the S3 implementation of the usecase.ExportObjectStorage
// port.
//
// [Ja] S3ExportStorage は usecase.ExportObjectStorage port の S3 実装。
type S3ExportStorage struct {
	client *s3.Client

	// feature/s3/manager is deprecated in favor of feature/s3/transfermanager,
	// but the successor is still a v0 preview module with no API stability
	// guarantee. Keep the stable v1 manager for this reliability-sensitive
	// path and re-evaluate once transfermanager reaches v1.
	//
	// [Ja] feature/s3/manager は feature/s3/transfermanager への移行が案内され
	// deprecated だが、後継はまだ v0 のプレビューモジュールで API 安定性の保証が
	// ない。信頼性が求められる本経路では安定版 v1 の manager を使い続け、
	// transfermanager が v1 になった時点で再評価する。
	uploader *manager.Uploader //nolint:staticcheck // SA1019: see the comment above
	bucket   string
}

// NewS3ExportStorage builds an S3 client and a multipart uploader for the
// configured bucket.
//
// [Ja] NewS3ExportStorage は設定されたバケット用の S3 クライアントと
// multipart アップローダーを構築する。
func NewS3ExportStorage(cfg S3Config) *S3ExportStorage {
	awsCfg := aws.Config{
		Region: cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		),
		BaseEndpoint: aws.String(cfg.Endpoint),
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// Path-style addressing keeps the bucket name in the URL path instead
		// of the hostname. R2 supports it, and it also works with endpoints
		// that have no per-bucket DNS (local S3-compatible servers).
		//
		// [Ja] パス形式アドレッシングはバケット名をホスト名ではなく URL パスに
		// 置く。R2 はこれをサポートしており、バケットごとの DNS を持たない
		// エンドポイント (ローカルの S3 互換サーバー) でも動作する。
		o.UsePathStyle = true
	})

	return &S3ExportStorage{
		client: client,
		// The uploader streams the body in bounded part-size chunks.
		// LeavePartsOnError disables the uploader's built-in abort: it reuses
		// the upload context, so a cancellation-induced failure would cancel
		// the abort as well and its error would be dropped silently, leaving
		// billable orphan parts behind. Upload aborts explicitly with a
		// detached cleanup context instead.
		//
		// [Ja] アップローダーは body を一定サイズのチャンク単位でストリーミング
		// する。LeavePartsOnError でアップローダー組み込みの abort を無効化する。
		// 組み込みの abort は upload と同じ context を使い回すため、キャンセル
		// 起因の失敗では abort 自体もキャンセルされてエラーは黙って捨てられ、
		// 課金対象の孤児パートが残ってしまう。代わりに Upload が切り離した
		// cleanup context で明示的に abort する。
		uploader: manager.NewUploader(client, func(u *manager.Uploader) { //nolint:staticcheck // SA1019: see the uploader field comment
			u.LeavePartsOnError = true
		}),
		bucket: cfg.BucketName,
	}
}

// Upload streams body to the bucket under key via multipart upload. The whole
// body is never buffered in memory; only bounded part buffers are held. When
// a started multipart upload fails, the uploaded parts are aborted with a
// detached cleanup context, even when the failure cause is the cancellation
// of ctx itself.
//
// [Ja] Upload は multipart upload で body を key の位置へストリーミング
// アップロードする。body 全体はメモリに保持せず、一定サイズのパートバッファ
// のみを保持する。開始済みの multipart upload が失敗した場合は、失敗原因が
// ctx 自体のキャンセルであっても、切り離した cleanup context でアップロード
// 済みパートを中断する。
func (s *S3ExportStorage) Upload(ctx context.Context, key string, body io.Reader) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{ //nolint:staticcheck // SA1019: see the uploader field comment
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(key),
		Body:         body,
		ContentType:  aws.String(exportContentType),
		CacheControl: aws.String(exportCacheControl),
	})
	if err != nil {
		if abortErr := s.abortMultipartUpload(ctx, key, err); abortErr != nil {
			// Keep both the upload error and the abort error so neither
			// failure is lost.
			//
			// [Ja] upload エラーと abort エラーのどちらも失われないよう
			// 両方を保持する。
			err = errors.Join(err, abortErr)
		}
		return fmt.Errorf("オブジェクトのアップロードに失敗 (key: %s): %w", key, err)
	}
	return nil
}

// abortMultipartUpload aborts the multipart upload left behind by a failed
// Upload call, and is a no-op when the failure did not reach the multipart
// path (single PutObject). It runs on a context detached from the upload
// context so the abort still executes when the upload failed because ctx was
// canceled.
//
// [Ja] abortMultipartUpload は失敗した Upload が残した multipart upload を
// 中断する。multipart 経路に到達していない失敗 (単発 PutObject) では何もしない。
// upload の context から切り離した context で実行することで、ctx のキャンセルが
// 失敗原因の場合でも abort を実行できるようにする。
func (s *S3ExportStorage) abortMultipartUpload(ctx context.Context, key string, uploadErr error) error {
	var multiErr manager.MultiUploadFailure //nolint:staticcheck // SA1019: see the uploader field comment
	if !errors.As(uploadErr, &multiErr) {
		return nil
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), abortMultipartUploadTimeout)
	defer cancel()

	if _, err := s.client.AbortMultipartUpload(cleanupCtx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(multiErr.UploadID()),
	}); err != nil {
		return fmt.Errorf("multipart upload の中断に失敗 (key: %s): %w", key, err)
	}
	return nil
}

// Download returns the object body as a stream together with its size in
// bytes. The caller must close the returned io.ReadCloser; closing it also
// releases the underlying HTTP response body.
//
// [Ja] Download はオブジェクト本体のストリームとバイト単位のサイズを返す。
// 返された io.ReadCloser は呼び出し側が必ず閉じる。閉じると背後の HTTP
// レスポンスボディも解放される。
func (s *S3ExportStorage) Download(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("オブジェクトのダウンロードに失敗 (key: %s): %w", key, err)
	}
	return out.Body, aws.ToInt64(out.ContentLength), nil
}

// Delete removes the object stored under key. A missing object is treated as
// success: the storage may answer 404 for a key that is already gone, and
// treating that as success keeps retried cleanup jobs idempotent.
//
// [Ja] Delete は key の位置のオブジェクトを削除する。オブジェクトが存在しない
// 場合は成功として扱う。ストレージは削除済みのキーに 404 を返すことがあり、
// これを成功扱いにすることでリトライされる cleanup ジョブが冪等になる。
func (s *S3ExportStorage) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		if isNotFoundErr(err) {
			return nil
		}
		return fmt.Errorf("オブジェクトの削除に失敗 (key: %s): %w", key, err)
	}
	return nil
}

// ListPrefix visits every object under prefix that sorts strictly after
// startAfter, passing its key and last-modified time to yield. It follows the
// ListObjectsV2 pagination until the listing is exhausted so callers never
// receive a silently truncated result.
//
// startAfter is sent on the first request only. ListObjectsV2 takes it as the
// key to begin after, but a request that also carries a continuation token
// resumes from the token instead, and the token already encodes the position
// the walk reached.
//
// [Ja] ListPrefix は prefix 配下で startAfter より厳密に後ろへ並ぶ全オブジェクトを
// 走査し、key と最終更新時刻を yield に渡す。ListObjectsV2 のページングを最後まで
// 辿り、呼び出し側が黙って切り詰められた結果を受け取ることがないようにする。
//
// startAfter を送るのは最初のリクエストだけである。ListObjectsV2 はこれを「この key の
// 次から開始する」として扱うが、continuation token を併せて持つリクエストは token の
// 位置から再開し、token は走査が到達した位置をすでに表しているため。
func (s *S3ExportStorage) ListPrefix(
	ctx context.Context,
	prefix, startAfter string,
	yield func(key string, lastModified time.Time) error,
) error {
	var continuationToken *string
	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		}
		if continuationToken == nil && startAfter != "" {
			input.StartAfter = aws.String(startAfter)
		}

		out, err := s.client.ListObjectsV2(ctx, input)
		if err != nil {
			return fmt.Errorf("オブジェクト一覧の取得に失敗 (prefix: %s): %w", prefix, err)
		}
		for _, obj := range out.Contents {
			if obj.Key == nil || obj.LastModified == nil {
				return fmt.Errorf("オブジェクト一覧の応答が不正 (prefix: %s)", prefix)
			}
			if err := yield(*obj.Key, *obj.LastModified); err != nil {
				return fmt.Errorf("オブジェクト一覧の処理に失敗 (prefix: %s): %w", prefix, err)
			}
		}
		if !aws.ToBool(out.IsTruncated) {
			return nil
		}
		// A truncated listing without a continuation token would reset the
		// loop to the first page and repeat forever. Well-behaved S3 / R2
		// never do this, but guard against it so a malformed response fails
		// the reconciliation job instead of hanging it.
		//
		// [Ja] 切り詰められた一覧なのに continuation token が無い応答は、ループを
		// 1 ページ目に戻して永久に繰り返してしまう。仕様に従う S3 / R2 では起き
		// ないが、不正な応答がリコンシリエーションジョブをハングさせず失敗させる
		// よう防御する。
		if out.NextContinuationToken == nil {
			return fmt.Errorf("オブジェクト一覧の応答が不正 (prefix: %s)", prefix)
		}
		continuationToken = out.NextContinuationToken
	}
}

// isNotFoundErr reports whether err is the S3 NoSuchKey error, the only
// failure that means the object is already gone. Other errors, including a
// NoSuchBucket 404, are real failures that must not be swallowed as success.
//
// [Ja] isNotFoundErr は err が S3 の NoSuchKey エラー (オブジェクトが既に
// 存在しないことを意味する唯一の失敗) かどうかを返す。NoSuchBucket の 404 を
// 含むそれ以外のエラーは、成功として握り潰してはならない本物の失敗。
func isNotFoundErr(err error) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey"
}
