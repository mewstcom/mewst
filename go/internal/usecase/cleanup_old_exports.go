package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// CleanupOldExportsTimeoutは掃除1回の実行の上限。通常の実行が削除するのは
// エクスポート1件のため、これは要求を受け付けたまま応答しなくなったストレージに
// 対する歯止めである。この掃除はタイムライン配信やメール送信と既定キューを共有して
// おり、上限の無い実行はそれらのworkerの1つを、プロセスが生きている限り占有する。
const CleanupOldExportsTimeout = 5 * time.Minute

// CleanupOldExportsUsecaseは、プロフィールの置き換え済みsucceeded export
// (最新のsucceeded以外のすべて) を削除する。保持ポリシーはプロフィールごとに
// アーカイブ1件を保つもので、成功は前の成功からダウンロードを引き継ぐだけである。
// 置き換えられたものを取り除くのが本UseCaseの仕事になる。
//
// 候補はジョブ引数で運ぶのではなく、クエリのたびにexportsテーブルから導出する。
// 実行中に成功が確定すると、それが置き換えたエクスポートも候補になるため、残すものを
// 名指しする引数はその時点で古くなっている。導出することで、現在のダウンロード対象を
// 保護する仕組みも要らなくなる。最新の成功はそもそも一覧に現れない。
type CleanupOldExportsUsecase struct {
	exportRepo    *repository.ExportRepository
	objectStorage ExportObjectStorage
}

// NewCleanupOldExportsUsecaseはCleanupOldExportsUsecaseを生成する。
func NewCleanupOldExportsUsecase(
	exportRepo *repository.ExportRepository,
	objectStorage ExportObjectStorage,
) *CleanupOldExportsUsecase {
	return &CleanupOldExportsUsecase{
		exportRepo:    exportRepo,
		objectStorage: objectStorage,
	}
}

// Executeはプロフィールの最新succeededより古いsucceeded exportを、
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

	// 削除した内容はエラーを返す前に記録する。失敗で終わる実行でも、そこへ至る
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
