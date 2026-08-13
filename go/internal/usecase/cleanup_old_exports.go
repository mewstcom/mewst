package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// CleanupOldExportsTimeout bounds one cleanup run. The usual run deletes a
// single export, so this is the backstop for a storage that accepted the
// request and then stopped answering: the cleanup shares the default queue with
// the timeline delivery and the emails, and a run without a bound would hold
// one of those workers for as long as the process lives.
//
// [Ja] CleanupOldExportsTimeout は掃除 1 回の実行の上限。通常の実行が削除するのは
// エクスポート 1 件のため、これは要求を受け付けたまま応答しなくなったストレージに
// 対する歯止めである。この掃除はタイムライン配信やメール送信と既定キューを共有して
// おり、上限の無い実行はそれらの worker の 1 つを、プロセスが生きている限り占有する。
const CleanupOldExportsTimeout = 5 * time.Minute

// CleanupOldExportsUsecase deletes a profile's superseded succeeded exports:
// every succeeded export except its latest one. The retention policy keeps one
// archive per profile, and a success only takes the download over from the
// previous one — removing what it replaced is this use case's work.
//
// The candidates are re-derived from the exports table on every query rather
// than carried in the job's arguments. A success that lands while a run is in
// progress makes the export it replaced a candidate too, and an argument
// naming what to keep would already be out of date by then. Deriving also
// leaves nothing to protect the current download from: the latest success is
// never in the list to begin with.
//
// [Ja] CleanupOldExportsUsecase は、プロフィールの置き換え済み succeeded export
// (最新の succeeded 以外のすべて) を削除する。保持ポリシーはプロフィールごとに
// アーカイブ 1 件を保つもので、成功は前の成功からダウンロードを引き継ぐだけである。
// 置き換えられたものを取り除くのが本 UseCase の仕事になる。
//
// 候補はジョブ引数で運ぶのではなく、クエリのたびに exports テーブルから導出する。
// 実行中に成功が確定すると、それが置き換えたエクスポートも候補になるため、残すものを
// 名指しする引数はその時点で古くなっている。導出することで、現在のダウンロード対象を
// 保護する仕組みも要らなくなる。最新の成功はそもそも一覧に現れない。
type CleanupOldExportsUsecase struct {
	exportRepo    *repository.ExportRepository
	objectStorage ExportObjectStorage
}

// NewCleanupOldExportsUsecase creates a CleanupOldExportsUsecase.
//
// [Ja] NewCleanupOldExportsUsecase は CleanupOldExportsUsecase を生成する。
func NewCleanupOldExportsUsecase(
	exportRepo *repository.ExportRepository,
	objectStorage ExportObjectStorage,
) *CleanupOldExportsUsecase {
	return &CleanupOldExportsUsecase{
		exportRepo:    exportRepo,
		objectStorage: objectStorage,
	}
}

// Execute deletes every succeeded export of the profile older than its latest
// success, object first and row second.
//
// A profile with nothing to clean up finishes without error, which is what the
// run enqueued after a profile's first success does.
//
// [Ja] Execute はプロフィールの最新 succeeded より古い succeeded export を、
// オブジェクト → 行の順にすべて削除する。
//
// 掃除するものが無いプロフィールはエラーなしで完了する。プロフィールの最初の成功の
// 後に投入された実行がこれにあたる。
func (uc *CleanupOldExportsUsecase) Execute(ctx context.Context, profileID model.ProfileID) error {
	deleted, err := deleteExportsInPages(ctx, uc.objectStorage, uc.exportRepo,
		func(ctx context.Context, pageSize int32) ([]*model.Export, error) {
			return uc.exportRepo.ListOldSucceededByProfileID(ctx, profileID, pageSize)
		},
	)

	// What was deleted is logged before any error is returned, so a run that
	// ends on a failure still records what it removed before reaching it.
	//
	// [Ja] 削除した内容はエラーを返す前に記録する。失敗で終わる実行でも、そこへ至る
	// までに削除したものが残るようにするため。
	if deleted > 0 {
		slog.InfoContext(ctx, "旧エクスポートを削除しました",
			"profile_id", profileID.String(),
			"deleted_count", deleted,
		)
	}
	if err != nil {
		return fmt.Errorf("旧エクスポートの削除に失敗 (profile_id: %s): %w", profileID.String(), err)
	}
	return nil
}
