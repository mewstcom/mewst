package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// CreateExportUsecaseはログイン中プロフィールのエクスポートを開始する。
// 申請をqueuedの行として永続化し、その後ジョブキューへ生成を依頼する。
//
// 申請の永続的な記録はジョブではなく行のほうである。queuedの行とRiverのジョブは
// 原子的に書けない (ジョブキューは自身のコネクションで動く) ため、先にINSERTを
// commitし、ジョブはその後に投入する。投入が失敗してもqueuedの行は残り、
// リコンシリエーションが引き受ける。投入の失敗でリクエストを失敗させないのは
// このためである。
type CreateExportUsecase struct {
	db              *sql.DB
	userProfileRepo *repository.UserProfileRepository
	exportRepo      *repository.ExportRepository
	dispatcher      *dispatcher.Dispatcher
	storageReady    bool
}

// NewCreateExportUsecaseはCreateExportUsecaseを生成する。storageReadyは
// エクスポート系Workerとエクスポート画面と同じreadinessで、合成時に一度だけ
// 解決した値である。
func NewCreateExportUsecase(
	db *sql.DB,
	userProfileRepo *repository.UserProfileRepository,
	exportRepo *repository.ExportRepository,
	d *dispatcher.Dispatcher,
	storageReady bool,
) *CreateExportUsecase {
	return &CreateExportUsecase{
		db:              db,
		userProfileRepo: userProfileRepo,
		exportRepo:      exportRepo,
		dispatcher:      d,
		storageReady:    storageReady,
	}
}

// CreateExportInputはエクスポート開始の入力パラメータ。UserIDとProfileIDは
// エクスポート画面と同じく認可を判断する組である。ActorIDは行に記録する申請者で
// あり、認可の判断には加わらない。
type CreateExportInput struct {
	UserID    model.UserID
	ProfileID model.ProfileID
	ActorID   model.ActorID
}

// CreateExportOutputはエクスポート開始の結果。Exportは作成されたqueuedの
// 行である。
type CreateExportOutput struct {
	Export *model.Export
}

// Executeはリクエストを認可し、queuedのエクスポートを永続化し、ジョブキューへ
// 生成を依頼する。
//
// 認可を利用可否の判定より先に行う理由はエクスポート画面と同じで、ユーザーが所有して
// いないプロフィールを、そのデプロイがエクスポートを実行できるかどうかに関わらず
// 同じ形で拒否するためである。実行できないデプロイは、何かを書き込む前に拒否される
// ため、生成するWorkerが存在しない行が作られることはない。
func (uc *CreateExportUsecase) Execute(ctx context.Context, input CreateExportInput) (*CreateExportOutput, error) {
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

	export, err := uc.createExport(ctx, input)
	if err != nil {
		return nil, err
	}

	uc.enqueueGeneration(ctx, export.ID)

	return &CreateExportOutput{Export: export}, nil
}

// authorizeはログイン中ユーザーが対象プロフィールを現在所有していることを
// 確認する。条件はGetExportShowUsecaseと同じで、プロフィールのポストを
// エクスポートする権利は、誰が申請したかを記録するactor行ではなく、そのユーザーが
// 今持っている所有関係から生じる。所有していないプロフィールはnot foundとして
// 拒否し、応答から既存のプロフィールと存在しないプロフィールを区別できないようにする。
func (uc *CreateExportUsecase) authorize(ctx context.Context, input CreateExportInput) error {
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

// createExportはプロフィールのfailedなエクスポートの削除と、新しいqueuedの
// エクスポートの挿入を1つのtransactionで行う。
//
// 2つの変更は同じcommitに属する。failedの行が不要になるのは新しい申請がそれを
// 置き換えるからであり、プロフィールの1つしかない実行枠を取り損ねたcreateは、
// その行を残さなければならない。まとめてrollbackすることで保持の上限も保たれる。
// succeededでも進行中でもないエクスポートをプロフィールが2件持つことがなくなる
// ためである。
func (uc *CreateExportUsecase) createExport(ctx context.Context, input CreateExportInput) (*model.Export, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	exportRepo := uc.exportRepo.WithTx(tx)

	if _, err := exportRepo.DeleteFailedByProfileID(ctx, input.ProfileID); err != nil {
		return nil, fmt.Errorf("失敗したエクスポートの削除に失敗: %w", err)
	}

	export, err := exportRepo.Create(ctx, repository.CreateExportInput{
		ProfileID: input.ProfileID,
		ActorID:   input.ActorID,
	})
	if err != nil {
		if errors.Is(err, repository.ErrActiveExportExists) {
			return nil, &model.AppError{
				Code:     model.AppErrCodeConflict,
				UserMsg:  i18n.T(ctx, "flash_export_already_in_progress"),
				Internal: err,
				Metadata: map[string]string{"profile_id": input.ProfileID.String()},
			}
		}
		return nil, fmt.Errorf("エクスポートの作成に失敗: %w", err)
	}
	// 行が無いのは、プロフィールが削除処理の確立する境界を越えた場合である。
	// 失敗ではないため、エラーではなく拒否として扱う。エクスポートがアーカイブする
	// はずのポストは、これから消えていくものであるため。
	if export == nil {
		return nil, &model.AppError{
			Code:     model.AppErrCodeConflict,
			UserMsg:  i18n.T(ctx, "flash_export_profile_deleting"),
			Internal: errors.New("プロフィールの削除が開始済みのためエクスポートを作成しない"),
			Metadata: map[string]string{"profile_id": input.ProfileID.String()},
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return export, nil
}

// enqueueGenerationはcommit済みのエクスポートの生成をジョブキューへ依頼する。
// 失敗はログに記録して握りつぶす。queuedの行がdurable work intentであり、ジョブを
// 得られなかったqueuedのエクスポートにはリコンシリエーションがジョブを投入するため、
// ここでリクエストを失敗として報告すると、実際には実行されるエクスポートを拒否する
// ことになる。
func (uc *CreateExportUsecase) enqueueGeneration(ctx context.Context, exportID model.ExportID) {
	if _, err := uc.dispatcher.EnqueueGenerateExport(ctx, exportID.String()); err != nil {
		slog.WarnContext(ctx, "エクスポート生成ジョブの投入に失敗", "error", err, "export_id", exportID.String())
	}
}
