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

// exportArchiveFileNameDateLayoutはアーカイブのファイル名に入れる生成日の
// 書式。ファイル名はダウンロード後にファイルアプリが表示するものであるため、
// 時刻ではなく暦日を持たせる。
const exportArchiveFileNameDateLayout = "20060102"

// GetExportDownloadUsecaseはログイン中ユーザーに対して、プロフィールの最新の
// 成功したエクスポートのzipを開く。
//
// ダウンロード対象はリクエストが指定する値ではなく、プロフィールの現在の状態から
// 解決する。保持ポリシーによりプロフィールがダウンロードできるアーカイブは最大1件
// であるため、古くなった画面から辿られたリンクでも、既に置き換えられたものではなく
// 今存在するアーカイブが得られる。
type GetExportDownloadUsecase struct {
	userProfileRepo *repository.UserProfileRepository
	userRepo        *repository.UserRepository
	exportRepo      *repository.ExportRepository
	storage         ExportObjectStorage
	storageReady    bool
}

// NewGetExportDownloadUsecaseはGetExportDownloadUsecaseを生成する。
// storageReadyはエクスポート画面・エクスポート開始と同じreadinessで、storageは
// その同じ値から構築したオブジェクトストレージである。readinessがfalseのデプロイ
// には渡せるオブジェクトストレージが無いため、storageへ到達する前にreadinessを
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

// GetExportDownloadInputはエクスポートのダウンロードの入力パラメータ。
// UserIDとProfileIDは、エクスポート画面・エクスポート開始と同じく認可を判断する
// 組である。
type GetExportDownloadInput struct {
	UserID    model.UserID
	ProfileID model.ProfileID
}

// GetExportDownloadOutputは開いたアーカイブ。Bodyはオブジェクトのストリームで
// 呼び出し側が必ず閉じる。Sizeはバイト単位の長さ、FileNameは提供する際の名前である。
// SizeとFileNameはレスポンスを説明するためにHandlerが必要とするメタデータであり、
// ここで解決することでHandlerをその判断から切り離す。
type GetExportDownloadOutput struct {
	Body     io.ReadCloser
	Size     int64
	FileName string
}

// Executeはログイン中ユーザーの対象プロフィールに対する認可を行い、
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
	// 成功したエクスポートを持たないプロフィールには渡すものが無い。画面が
	// ダウンロードリンクを描画するのは1件存在する間だけであるため、ここへ到達した
	// リクエストは、そのプロフィールを既に説明していないリンクを辿ったか、リンクを
	// 経ずに来たものである。所有していないプロフィールと同じ条件でnot foundとして
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
		return nil, fmt.Errorf("成功したエクスポートにobject keyが無い (export_id: %s)", export.ID.String())
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

// authorizeはログイン中ユーザーが対象プロフィールを現在所有していることを
// 確認する。条件はGetExportShowUsecase・CreateExportUsecaseと同じで、プロフィールの
// ポストのアーカイブを得る権利は、そのユーザーが今持っている所有関係から生じる。
// 所有していないプロフィールはnot foundとして拒否し、応答から既存のプロフィールと
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

// archiveFileNameはアーカイブを提供する際の名前
// (mewst-export-{YYYYMMDD}.zip) を組み立てる。日付はダウンロードするユーザーの
// タイムゾーンで解釈する。
//
// UTCではなく読み手自身のゾーンで読むのは、ファイル名がアーカイブについてファイル
// アプリが示す唯一の日付であり、アーカイブ自身のページと同じ日を指すべきであるため。
// 現時点でプロフィールを所有し、ここへ到達できるユーザーは1人であり、その読み手は
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

// exportGeneratedAtはエクスポートが完了した時刻を返し、無い場合は申請された
// 時刻へフォールバックする。成功したエクスポートは常にfinished_atを持つ (状態の
// CHECK制約が要求する)。フォールバックは、何らかの理由でそれを失った行にも実際に
// 属する日を与えるためのもので、無傷のアーカイブのダウンロードを失敗させるよりよい。
func exportGeneratedAt(export *model.Export) time.Time {
	if export.FinishedAt != nil {
		return *export.FinishedAt
	}
	return export.CreatedAt
}
