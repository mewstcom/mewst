// Package storageはエクスポート機能で使うS3互換オブジェクトストレージ
// (Cloudflare R2) へのInfrastructure層アダプタを提供する。
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
	// exportContentTypeとexportCacheControlはオブジェクトのメタデータ
	// として保存し、アプリ外で取得された場合も安全な既定値が付くようにする。
	// ダウンロード時のHTTPレスポンスヘッダーはGoサーバーが必ず上書きする。
	exportContentType  = "application/zip"
	exportCacheControl = "private, no-store"

	// abortMultipartUploadTimeoutは失敗したmultipart uploadを中断する
	// cleanup呼び出しの上限時間。cleanup contextはuploadのcontextから
	// 切り離すため、このtimeoutがabortのハングを防ぐ。
	abortMultipartUploadTimeout = 30 * time.Second
)

// S3ConfigはS3互換オブジェクトストレージへの接続設定を保持する。
// MEWST_S3_* 環境変数と対応し、呼び出し側で値を詰め替えることで本パッケージを
// internal/configから独立させる。
type S3Config struct {
	BucketName      string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

// S3ExportStorageはusecase.ExportObjectStorage portのS3実装。
type S3ExportStorage struct {
	client *s3.Client

	// feature/s3/managerはfeature/s3/transfermanagerへの移行が案内され
	// deprecatedだが、後継はまだv0のプレビューモジュールでAPI安定性の保証が
	// ない。信頼性が求められる本経路では安定版v1のmanagerを使い続け、
	// transfermanagerがv1になった時点で再評価する。
	uploader *manager.Uploader //nolint:staticcheck // SA1019: see the comment above
	bucket   string
}

// NewS3ExportStorageは設定されたバケット用のS3クライアントと
// multipartアップローダーを構築する。
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
		// パス形式アドレッシングはバケット名をホスト名ではなくURLパスに
		// 置く。R2はこれをサポートしており、バケットごとのDNSを持たない
		// エンドポイント (ローカルのS3互換サーバー) でも動作する。
		o.UsePathStyle = true
	})

	return &S3ExportStorage{
		client: client,
		// アップローダーはbodyを一定サイズのチャンク単位でストリーミング
		// する。LeavePartsOnErrorでアップローダー組み込みのabortを無効化する。
		// 組み込みのabortはuploadと同じcontextを使い回すため、キャンセル
		// 起因の失敗ではabort自体もキャンセルされてエラーは黙って捨てられ、
		// 課金対象の孤児パートが残ってしまう。代わりにUploadが切り離した
		// cleanup contextで明示的にabortする。
		uploader: manager.NewUploader(client, func(u *manager.Uploader) { //nolint:staticcheck // SA1019: see the uploader field comment
			u.LeavePartsOnError = true
		}),
		bucket: cfg.BucketName,
	}
}

// Uploadはmultipart uploadでbodyをkeyの位置へストリーミング
// アップロードする。body全体はメモリに保持せず、一定サイズのパートバッファ
// のみを保持する。開始済みのmultipart uploadが失敗した場合は、失敗原因が
// ctx自体のキャンセルであっても、切り離したcleanup contextでアップロード
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
			// uploadエラーとabortエラーのどちらも失われないよう
			// 両方を保持する。
			err = errors.Join(err, abortErr)
		}
		return fmt.Errorf("オブジェクトのアップロードに失敗 (key: %s): %w", key, err)
	}
	return nil
}

// abortMultipartUploadは失敗したUploadが残したmultipart uploadを
// 中断する。multipart経路に到達していない失敗 (単発PutObject) では何もしない。
// uploadのcontextから切り離したcontextで実行することで、ctxのキャンセルが
// 失敗原因の場合でもabortを実行できるようにする。
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
		return fmt.Errorf("multipart uploadの中断に失敗 (key: %s): %w", key, err)
	}
	return nil
}

// Downloadはオブジェクト本体のストリームとバイト単位のサイズを返す。
// 返されたio.ReadCloserは呼び出し側が必ず閉じる。閉じると背後のHTTP
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

// Deleteはkeyの位置のオブジェクトを削除する。オブジェクトが存在しない
// 場合は成功として扱う。ストレージは削除済みのキーに404を返すことがあり、
// これを成功扱いにすることでリトライされるcleanupジョブが冪等になる。
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

// ListPrefixはprefix配下でstartAfterより厳密に後ろへ並ぶ全オブジェクトを
// 走査し、keyと最終更新時刻をyieldに渡す。ListObjectsV2のページングを最後まで
// 辿り、呼び出し側が黙って切り詰められた結果を受け取ることがないようにする。
//
// startAfterを送るのは最初のリクエストだけである。ListObjectsV2はこれを「このkeyの
// 次から開始する」として扱うが、continuation tokenを併せて持つリクエストはtokenの
// 位置から再開し、tokenは走査が到達した位置をすでに表しているため。
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
		// 切り詰められた一覧なのにcontinuation tokenが無い応答は、ループを
		// 1ページ目に戻して永久に繰り返してしまう。仕様に従うS3 / R2では起き
		// ないが、不正な応答がリコンシリエーションジョブをハングさせず失敗させる
		// よう防御する。
		if out.NextContinuationToken == nil {
			return fmt.Errorf("オブジェクト一覧の応答が不正 (prefix: %s)", prefix)
		}
		continuationToken = out.NextContinuationToken
	}
}

// isNotFoundErrはerrがS3のNoSuchKeyエラー (オブジェクトが既に
// 存在しないことを意味する唯一の失敗) かどうかを返す。NoSuchBucketの404を
// 含むそれ以外のエラーは、成功として握り潰してはならない本物の失敗。
func isNotFoundErr(err error) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey"
}
