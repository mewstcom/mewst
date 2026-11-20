package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// DeleteExportsForProfileUsecaseはプロフィールのエクスポートをstatusを
// 問わずすべて削除し、プロフィール自体を削除できる状態にする。exportsの外部キーが
// CASCADEではなくON DELETE NO ACTIONなのは、エクスポートが行とオブジェクト
// ストレージ上のオブジェクトの両方であり、CASCADEでは行だけが消えてオブジェクトを
// 指すものが無くなるためである。したがってDBはエクスポートが残るプロフィールの
// 削除を拒否し、その拒否を解消するためにプロフィールの削除処理が最初に実行するのが
// 本UseCaseである。
//
// 併せてプロフィールの送信待ち完了メールも取り消す。それらの行がexportより長く残る
// のは意図した設計で、保持cleanupが旧エクスポートを削除しても未送信のメールを失わ
// ないようにするためである。削除されるプロフィールに送るべきメールは無い。
//
// 意図的にジョブで駆動しない。プロフィールの削除は、行を消す前にそのエクスポートが
// 消えたことを知る必要があるため、呼び出し側は結果を確認できないジョブを投入するので
// はなく、これを実行して完了を待つ。
type DeleteExportsForProfileUsecase struct {
	exportRepo       *repository.ExportRepository
	notificationRepo *repository.ExportCompletionNotificationRepository
	deletionGuard    ExportProfileDeletionGuard
	objectStorage    ExportObjectStorage
}

// NewDeleteExportsForProfileUsecaseはDeleteExportsForProfileUsecaseを
// 生成する。
func NewDeleteExportsForProfileUsecase(
	exportRepo *repository.ExportRepository,
	notificationRepo *repository.ExportCompletionNotificationRepository,
	deletionGuard ExportProfileDeletionGuard,
	objectStorage ExportObjectStorage,
) *DeleteExportsForProfileUsecase {
	return &DeleteExportsForProfileUsecase{
		exportRepo:       exportRepo,
		notificationRepo: notificationRepo,
		deletionGuard:    deletionGuard,
		objectStorage:    objectStorage,
	}
}

// Executeはプロフィールのエクスポートをオブジェクト → 行の順にすべて削除し、
// 送信待ちの完了メールを取り消し、失敗した場合はそれを報告して、呼び出し側が
// プロフィールを削除する前に止まれるようにする。
//
// 一覧取得前に削除マーカーを永続化し、プロフィールのexport用排他lockを取得する。
// 新しい作成・生成はマーカーで止まり、lockはすでにupload中だった生成を待つ。
// そのため、本UseCaseが削除した後に同じキーをuploadが再公開することはない。
//
// メールの取り消しはエクスポートの削除より前ではなく後に行う。通知を作成するのは
// succeededへの遷移であり、それにはexport行が必要である。プロフィールの行が1つも
// 残っていなければ新たな通知は現れず、取り消しは最終的なものになる。先に取り消すと、
// 実行中に成功した生成のためにその隙間が開いたままになる。
func (uc *DeleteExportsForProfileUsecase) Execute(ctx context.Context, profileID model.ProfileID) (err error) {
	release, found, err := uc.deletionGuard.BeginDeletion(ctx, profileID)
	if err != nil {
		return fmt.Errorf("プロフィールのexport削除guardの取得に失敗 (profile_id: %s): %w", profileID.String(), err)
	}
	if !found {
		slog.InfoContext(ctx, "エクスポート削除対象のプロフィールが見つかりません",
			"profile_id", profileID.String(),
		)
		return nil
	}
	defer func() {
		if releaseErr := release(); releaseErr != nil {
			err = errors.Join(err, fmt.Errorf("プロフィールのexport削除guardの解放に失敗: %w", releaseErr))
		}
	}()

	deleted, err := deleteExportsInPages(ctx, uc.objectStorage, uc.exportRepo,
		func(ctx context.Context, pageSize int32) ([]*model.Export, error) {
			return uc.exportRepo.ListByProfileID(ctx, profileID, pageSize)
		},
	)

	// 削除した内容はエラーを返す前に記録する。失敗で終わる実行でも、そこへ至る
	// までに削除したものが残るようにするため。
	if deleted > 0 {
		slog.InfoContext(ctx, "プロフィールのエクスポートを削除しました",
			"profile_id", profileID.String(),
			"deleted_count", deleted,
		)
	}
	if err != nil {
		return fmt.Errorf("プロフィールのエクスポートの削除に失敗 (profile_id: %s): %w", profileID.String(), err)
	}

	cancelled, err := uc.notificationRepo.DeleteByProfileID(ctx, profileID)
	if err != nil {
		return fmt.Errorf("プロフィールの完了通知の取り消しに失敗 (profile_id: %s): %w", profileID.String(), err)
	}
	if cancelled > 0 {
		slog.InfoContext(ctx, "プロフィールの送信待ち完了通知を取り消しました",
			"profile_id", profileID.String(),
			"cancelled_count", cancelled,
		)
	}
	return nil
}
