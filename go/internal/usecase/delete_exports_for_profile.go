package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// DeleteExportsForProfileUsecase removes every export of a profile, whatever
// its status, so that the profile itself can be deleted. The exports foreign
// key is ON DELETE NO ACTION rather than CASCADE: an export is a row and an
// object in the object storage, and a cascade would take the row while leaving
// the object with nothing naming it. The database therefore refuses to delete a
// profile that still has exports, and this use case is what a profile deletion
// runs first to satisfy that refusal.
//
// It also cancels the profile's pending completion emails. Those rows outlive
// the export on purpose, so that retention cleanup can delete an old export
// without losing the email it still owes; a profile being deleted owes none.
//
// It is deliberately not driven by a job. Deleting a profile has to know that
// its exports are gone before it removes the row, so the caller runs this and
// waits for it rather than enqueueing work it cannot see the outcome of.
//
// [Ja] DeleteExportsForProfileUsecase はプロフィールのエクスポートを status を
// 問わずすべて削除し、プロフィール自体を削除できる状態にする。exports の外部キーが
// CASCADE ではなく ON DELETE NO ACTION なのは、エクスポートが行とオブジェクト
// ストレージ上のオブジェクトの両方であり、CASCADE では行だけが消えてオブジェクトを
// 指すものが無くなるためである。したがって DB はエクスポートが残るプロフィールの
// 削除を拒否し、その拒否を解消するためにプロフィールの削除処理が最初に実行するのが
// 本 UseCase である。
//
// 併せてプロフィールの送信待ち完了メールも取り消す。それらの行が export より長く残る
// のは意図した設計で、保持 cleanup が旧エクスポートを削除しても未送信のメールを失わ
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

// NewDeleteExportsForProfileUsecase creates a DeleteExportsForProfileUsecase.
//
// [Ja] NewDeleteExportsForProfileUsecase は DeleteExportsForProfileUsecase を
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

// Execute deletes all of the profile's exports, object first and row second,
// cancels its pending completion emails, and reports a failure so that the
// caller stops before removing the profile.
//
// Before listing, it persists a deletion marker and takes the profile's
// exclusive export lock. New creates and generations stop at the marker, and
// the lock waits for a generation that was already uploading. Therefore no
// upload can publish the key again after this use case deletes it.
//
// The emails are cancelled after the exports rather than before. A notification
// is created by the transition into succeeded, which needs the export row, so
// once no row of the profile is left no further notification can appear and the
// cancellation is final. Cancelling first would leave that gap open for a
// generation that succeeds during the run.
//
// [Ja] Execute はプロフィールのエクスポートをオブジェクト → 行の順にすべて削除し、
// 送信待ちの完了メールを取り消し、失敗した場合はそれを報告して、呼び出し側が
// プロフィールを削除する前に止まれるようにする。
//
// 一覧取得前に削除マーカーを永続化し、プロフィールの export 用排他 lock を取得する。
// 新しい作成・生成はマーカーで止まり、lock はすでに upload 中だった生成を待つ。
// そのため、本 UseCase が削除した後に同じキーを upload が再公開することはない。
//
// メールの取り消しはエクスポートの削除より前ではなく後に行う。通知を作成するのは
// succeeded への遷移であり、それには export 行が必要である。プロフィールの行が 1 つも
// 残っていなければ新たな通知は現れず、取り消しは最終的なものになる。先に取り消すと、
// 実行中に成功した生成のためにその隙間が開いたままになる。
func (uc *DeleteExportsForProfileUsecase) Execute(ctx context.Context, profileID model.ProfileID) (err error) {
	release, found, err := uc.deletionGuard.BeginDeletion(ctx, profileID)
	if err != nil {
		return fmt.Errorf("プロフィールの export 削除 guard の取得に失敗 (profile_id: %s): %w", profileID.String(), err)
	}
	if !found {
		slog.InfoContext(ctx, "エクスポート削除対象のプロフィールが見つかりません",
			"profile_id", profileID.String(),
		)
		return nil
	}
	defer func() {
		if releaseErr := release(); releaseErr != nil {
			err = errors.Join(err, fmt.Errorf("プロフィールの export 削除 guard の解放に失敗: %w", releaseErr))
		}
	}()

	deleted, err := deleteExportsInPages(ctx, uc.objectStorage, uc.exportRepo,
		func(ctx context.Context, pageSize int32) ([]*model.Export, error) {
			return uc.exportRepo.ListByProfileID(ctx, profileID, pageSize)
		},
	)

	// What was deleted is logged before any error is returned, so a run that
	// ends on a failure still records what it removed before reaching it.
	//
	// [Ja] 削除した内容はエラーを返す前に記録する。失敗で終わる実行でも、そこへ至る
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
