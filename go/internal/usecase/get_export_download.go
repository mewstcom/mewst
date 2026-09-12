package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// exportArchiveFileNameDateLayout formats the date the export was generated on
// for the archive's file name. The name is what a file manager shows after the
// download, so it carries a calendar date rather than an instant.
//
// [Ja] exportArchiveFileNameDateLayout はアーカイブのファイル名に入れる生成日の
// 書式。ファイル名はダウンロード後にファイルアプリが表示するものであるため、
// 時刻ではなく暦日を持たせる。
const exportArchiveFileNameDateLayout = "20060102"

// GetExportDownloadUsecase opens the zip of the profile's latest succeeded
// export for the signed-in user.
//
// The download target is resolved from the profile's current state rather than
// from anything the request names: under the retention policy a profile has at
// most one downloadable archive, so a link followed from a page that has since
// gone stale still yields the archive that exists now instead of one that was
// already replaced.
//
// [Ja] GetExportDownloadUsecase はログイン中ユーザーに対して、プロフィールの最新の
// 成功したエクスポートの zip を開く。
//
// ダウンロード対象はリクエストが指定する値ではなく、プロフィールの現在の状態から
// 解決する。保持ポリシーによりプロフィールがダウンロードできるアーカイブは最大 1 件
// であるため、古くなった画面から辿られたリンクでも、既に置き換えられたものではなく
// 今存在するアーカイブが得られる。
type GetExportDownloadUsecase struct {
	userProfileRepo *repository.UserProfileRepository
	userRepo        *repository.UserRepository
	exportRepo      *repository.ExportRepository
	storage         ExportObjectStorage
	storageReady    bool
}

// NewGetExportDownloadUsecase creates a GetExportDownloadUsecase. storageReady
// is the same readiness the export page and the export start are gated on, and
// storage is the object storage built from that same resolved value; a
// deployment whose readiness is false has no object storage to pass, so the
// readiness is checked before storage is ever reached.
//
// [Ja] NewGetExportDownloadUsecase は GetExportDownloadUsecase を生成する。
// storageReady はエクスポート画面・エクスポート開始と同じ readiness で、storage は
// その同じ値から構築したオブジェクトストレージである。readiness が false のデプロイ
// には渡せるオブジェクトストレージが無いため、storage へ到達する前に readiness を
// 検査する。
func NewGetExportDownloadUsecase(
	userProfileRepo *repository.UserProfileRepository,
	userRepo *repository.UserRepository,
	exportRepo *repository.ExportRepository,
	storage ExportObjectStorage,
	storageReady bool,
) *GetExportDownloadUsecase {
	return &GetExportDownloadUsecase{
		userProfileRepo: userProfileRepo,
		userRepo:        userRepo,
		exportRepo:      exportRepo,
		storage:         storage,
		storageReady:    storageReady,
	}
}

// GetExportDownloadInput holds the input parameters for downloading an export.
// UserID and ProfileID are the pair authorization is decided on, exactly as on
// the export page and at the export start.
//
// [Ja] GetExportDownloadInput はエクスポートのダウンロードの入力パラメータ。
// UserID と ProfileID は、エクスポート画面・エクスポート開始と同じく認可を判断する
// 組である。
type GetExportDownloadInput struct {
	UserID    model.UserID
	ProfileID model.ProfileID
}

// GetExportDownloadOutput is the opened archive. Body is the object stream and
// the caller must close it, Size is its length in bytes and FileName is the
// name to offer it under. Size and FileName are metadata the Handler needs to
// describe the response, so resolving them here keeps the Handler free of the
// decisions behind them.
//
// [Ja] GetExportDownloadOutput は開いたアーカイブ。Body はオブジェクトのストリームで
// 呼び出し側が必ず閉じる。Size はバイト単位の長さ、FileName は提供する際の名前である。
// Size と FileName はレスポンスを説明するために Handler が必要とするメタデータであり、
// ここで解決することで Handler をその判断から切り離す。
type GetExportDownloadOutput struct {
	Body     io.ReadCloser
	Size     int64
	FileName string
}

// Execute authorizes the signed-in user against the target profile and opens
// the profile's downloadable archive.
//
// The steps are ordered so that nothing can fail after the stream is open: the
// object is the last thing reached, and everything handed to the caller
// alongside it is resolved first. No error path is therefore left with an open
// stream to abandon.
//
// [Ja] Execute はログイン中ユーザーの対象プロフィールに対する認可を行い、
// プロフィールのダウンロード可能なアーカイブを開く。
//
// ストリームを開いた後に失敗しうる処理が残らない順序にしている。オブジェクトは最後に
// 到達する対象で、それと併せて呼び出し側へ渡すものはすべて先に解決する。これにより、
// 開いたままのストリームを取り残すエラー経路が生まれない。
func (uc *GetExportDownloadUsecase) Execute(ctx context.Context, input GetExportDownloadInput) (*GetExportDownloadOutput, error) {
	if err := uc.authorize(ctx, input); err != nil {
		return nil, err
	}

	if !uc.storageReady {
		return nil, &model.AppError{
			Code:     model.AppErrCodeServiceUnavailable,
			UserMsg:  i18n.T(ctx, "error_export_unavailable"),
			Internal: errors.New("エクスポート用オブジェクトストレージが未設定"),
			Metadata: map[string]string{"profile_id": input.ProfileID.String()},
		}
	}

	export, err := uc.exportRepo.FindLatestSucceededByProfileID(ctx, input.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("最新の成功したエクスポートの取得に失敗: %w", err)
	}
	// A profile with no succeeded export has nothing to hand over. The page
	// renders the download link only while one exists, so a request that reaches
	// here followed a link that no longer describes the profile, or no link at
	// all. It is refused as not found, on the same terms as a profile the user
	// does not own.
	//
	// [Ja] 成功したエクスポートを持たないプロフィールには渡すものが無い。画面が
	// ダウンロードリンクを描画するのは 1 件存在する間だけであるため、ここへ到達した
	// リクエストは、そのプロフィールを既に説明していないリンクを辿ったか、リンクを
	// 経ずに来たものである。所有していないプロフィールと同じ条件で not found として
	// 拒否する。
	if export == nil {
		return nil, &model.AppError{
			Code:     model.AppErrCodeResourceNotFound,
			UserMsg:  i18n.T(ctx, "error_not_found_message"),
			Internal: errors.New("ダウンロードできるエクスポートが存在しない"),
			Metadata: map[string]string{"profile_id": input.ProfileID.String()},
		}
	}
	if export.ObjectKey == nil {
		return nil, fmt.Errorf("成功したエクスポートに object key が無い (export_id: %s)", export.ID.String())
	}

	fileName, err := uc.archiveFileName(ctx, input.UserID, export)
	if err != nil {
		return nil, err
	}

	body, size, err := uc.storage.Download(ctx, *export.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("エクスポートのオブジェクトの取得に失敗 (export_id: %s): %w", export.ID.String(), err)
	}

	return &GetExportDownloadOutput{
		Body:     body,
		Size:     size,
		FileName: fileName,
	}, nil
}

// authorize checks that the signed-in user currently owns the target profile,
// on the same terms as GetExportShowUsecase and CreateExportUsecase: the right
// to an archive of a profile's posts comes from the ownership the user holds
// right now. A profile the user does not own is refused as not found, so the
// response cannot be used to tell an existing profile from a missing one, nor
// an archive that exists from one that does not.
//
// [Ja] authorize はログイン中ユーザーが対象プロフィールを現在所有していることを
// 確認する。条件は GetExportShowUsecase・CreateExportUsecase と同じで、プロフィールの
// ポストのアーカイブを得る権利は、そのユーザーが今持っている所有関係から生じる。
// 所有していないプロフィールは not found として拒否し、応答から既存のプロフィールと
// 存在しないプロフィール、およびアーカイブの有無を区別できないようにする。
func (uc *GetExportDownloadUsecase) authorize(ctx context.Context, input GetExportDownloadInput) error {
	userProfile, err := uc.userProfileRepo.FindByProfileID(ctx, input.ProfileID)
	if err != nil {
		return fmt.Errorf("プロフィールの所有関係の取得に失敗: %w", err)
	}

	if userProfile == nil || userProfile.UserID != input.UserID {
		return &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	return nil
}

// archiveFileName builds the name the archive is offered under:
// mewst-export-{YYYYMMDD}.zip, dated in the downloading user's time zone.
//
// The date is read in the reader's own zone rather than in UTC because the file
// name is the only date a file manager shows for the archive, and it should
// name the same day the archive's own pages do. Today a profile is owned by the
// one user who can reach this, so that reader is also the one the archive was
// written for.
//
// [Ja] archiveFileName はアーカイブを提供する際の名前
// (mewst-export-{YYYYMMDD}.zip) を組み立てる。日付はダウンロードするユーザーの
// タイムゾーンで解釈する。
//
// UTC ではなく読み手自身のゾーンで読むのは、ファイル名がアーカイブについてファイル
// アプリが示す唯一の日付であり、アーカイブ自身のページと同じ日を指すべきであるため。
// 現時点でプロフィールを所有し、ここへ到達できるユーザーは 1 人であり、その読み手は
// アーカイブが書かれた対象の本人でもある。
func (uc *GetExportDownloadUsecase) archiveFileName(ctx context.Context, userID model.UserID, export *model.Export) (string, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("ユーザーの取得に失敗: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("ログイン中のユーザーが見つからない (user_id: %s)", userID.String())
	}

	generatedAt := exportGeneratedAt(export).In(exportLocation(ctx, user))
	return fmt.Sprintf("mewst-export-%s.zip", generatedAt.Format(exportArchiveFileNameDateLayout)), nil
}

// exportGeneratedAt returns the time the export finished, falling back to when
// it was requested. A succeeded export always carries finished_at (the state
// CHECK constraint requires it); the fallback names a row that somehow lost it
// after a day it does belong to, rather than failing the download of an archive
// that is otherwise intact.
//
// [Ja] exportGeneratedAt はエクスポートが完了した時刻を返し、無い場合は申請された
// 時刻へフォールバックする。成功したエクスポートは常に finished_at を持つ (状態の
// CHECK 制約が要求する)。フォールバックは、何らかの理由でそれを失った行にも実際に
// 属する日を与えるためのもので、無傷のアーカイブのダウンロードを失敗させるよりよい。
func exportGeneratedAt(export *model.Export) time.Time {
	if export.FinishedAt != nil {
		return *export.FinishedAt
	}
	return export.CreatedAt
}
