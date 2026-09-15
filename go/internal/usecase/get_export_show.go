package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// GetExportShowUsecaseはエクスポート画面 (GET /settings/export) がログイン中
// プロフィールについて表示する内容を取得する。状態メッセージを駆動する最新の
// エクスポートと、zipをダウンロードできるかを決める最新の成功したエクスポートを
// 読む。2つを分けて読むのは、より新しい進行中・失敗のエクスポートが、以前の
// エクスポートが既に作ったzipを隠さないようにするため。
type GetExportShowUsecase struct {
	userProfileRepo *repository.UserProfileRepository
	exportRepo      *repository.ExportRepository
	storageReady    bool
}

// NewGetExportShowUsecaseはGetExportShowUsecaseを生成する。storageReadyは
// そのデプロイでエクスポート用オブジェクトストレージが設定されているかを表し、
// エクスポート系Workerの登録と同じreadinessから合成時に一度だけ解決する。
func NewGetExportShowUsecase(
	userProfileRepo *repository.UserProfileRepository,
	exportRepo *repository.ExportRepository,
	storageReady bool,
) *GetExportShowUsecase {
	return &GetExportShowUsecase{
		userProfileRepo: userProfileRepo,
		exportRepo:      exportRepo,
		storageReady:    storageReady,
	}
}

// GetExportShowInputはエクスポート画面の取得の入力パラメータ。UserIDと
// ProfileIDはどちらもセッション由来で、エクスポート対象はプロフィール、
// ユーザーはそのプロフィールを現在所有していなければならない側を表す。
type GetExportShowInput struct {
	UserID    model.UserID
	ProfileID model.ProfileID
}

// GetExportShowOutputはエクスポート画面の取得結果。プロフィールが
// エクスポートを持たない場合、およびAvailableがfalseの場合は両方nilになる
// (利用できないデプロイではエクスポートの状態を一切表示しない)。
type GetExportShowOutput struct {
	LatestExport          *model.Export
	LatestSucceededExport *model.Export
	Available             bool
}

// Executeはログイン中ユーザーの対象プロフィールに対する認可を行い、
// プロフィールのエクスポート状態を読む。
//
// 認可を利用可否の判定より先に行うのは、ユーザーが所有していないプロフィールを、
// そのデプロイがエクスポートを実行できるかどうかに関わらず同じ形で拒否するため。
// エクスポートが利用できない場合はexport行を読まない。画面は機能が利用できない
// ことだけを表示するため、その値は描画内容を変えられないからである。
func (uc *GetExportShowUsecase) Execute(ctx context.Context, input GetExportShowInput) (*GetExportShowOutput, error) {
	if err := uc.authorize(ctx, input); err != nil {
		return nil, err
	}

	if !uc.storageReady {
		return &GetExportShowOutput{Available: false}, nil
	}

	latest, err := uc.exportRepo.FindLatestByProfileID(ctx, input.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("最新のエクスポートの取得に失敗: %w", err)
	}

	latestSucceeded, err := uc.exportRepo.FindLatestSucceededByProfileID(ctx, input.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("最新の成功したエクスポートの取得に失敗: %w", err)
	}

	return &GetExportShowOutput{
		LatestExport:          latest,
		LatestSucceededExport: latestSucceeded,
		Available:             true,
	}, nil
}

// authorizeはログイン中ユーザーが対象プロフィールを現在所有していることを
// 確認する。セッションのactorは根拠にしない。actor行は誰が何を申請したかの記録で
// あり、プロフィールのエクスポートを見る権利は、そのユーザーが今持っている所有関係
// から生じるため。グループ所有プロフィールの導入前は、その所有関係はuser_profilesの
// 関連付けである。
//
// 所有していないプロフィールはnot foundとして拒否し、応答から既存のプロフィールと
// 存在しないプロフィールを区別できないようにする。
func (uc *GetExportShowUsecase) authorize(ctx context.Context, input GetExportShowInput) error {
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
