package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// exportDeletionPageSizeは1回の削除クエリが一度に保持するエクスポート数。
// プロフィールが削除対象として持つエクスポートは通常1件のため、この値は普段の
// 実行のためのものではない。削除が失敗し続けて溜まったバックログを処理するときの
// クエリを有界にする。
const exportDeletionPageSize int32 = 100

// deleteExportsInPagesはlistが返すエクスポートを走査して削除し、listが空を
// 返すまで次のページを要求する。各ページは新しいクエリであるため、候補がテーブルの
// 現在の状態に依存する一覧 (最新の成功より古いエクスポート) は、実行開始時点で固定
// されるのではなく毎回導出し直される。
//
// 走査は最初の失敗で止まり、それを報告する。削除できなかった候補は一覧の先頭に残る
// ため、そこを飛ばして続けると以降のどのページも同じ行から始まり、走査が終わらなく
// なる。保持cleanupはジョブキューが再試行し、リコンシリエーションが再導出する。
// プロフィール削除ではエラーを呼び出し元へ返し、プロフィール行を削除する前に止める。
// その削除フローを再実行すると、同じ残存候補が再び処理される。これは失敗を数えて先へ
// 進む孤児回収とは逆である。孤児回収の走査は自身が削除しない一覧の上を進むため、
// ストレージが拒否する1件のオブジェクトが、その後ろのすべてを隠してしまう。
func deleteExportsInPages(
	ctx context.Context,
	objectStorage ExportObjectStorage,
	exportRepo *repository.ExportRepository,
	list func(ctx context.Context, pageSize int32) ([]*model.Export, error),
) (int, error) {
	deleted := 0

	for {
		exports, err := list(ctx, exportDeletionPageSize)
		if err != nil {
			return deleted, err
		}
		if len(exports) == 0 {
			return deleted, nil
		}

		for _, export := range exports {
			if err := deleteExportWithObject(ctx, objectStorage, exportRepo, export); err != nil {
				return deleted, err
			}
			deleted++
		}
	}
}

// deleteExportWithObjectはエクスポートのアーカイブをオブジェクトストレージ
// から削除し、その後に行を削除する。
//
// オブジェクトを先にするのは、そのオブジェクトを指し示すのが行だからである。行を
// 消した後で止まった実行は、どこからも指されないオブジェクトを残す。それを見つけ
// られるのは日次の孤児回収だけなので、次の回収まで保存され課金され続ける。逆の順序
// なら、実行が残すのはオブジェクトが既に無い行であり、その行は同じジョブの次の実行が
// 削除する。誰も把握していないコストが生じる瞬間が無い。
//
// キーはobject_keyを読むのではなく行から導出する。succeededのエクスポートでは
// 両者は一致する (成功の遷移が記録したのがまさにこの値であるため) が、成功しな
// かったエクスポートはキーを保存しておらず、それでも遷移より先にアップロードした
// 試行のオブジェクトを保持しうる。導出すれば両方を扱えるうえ、エクスポートのキー
// 規約の外にあるオブジェクトを指すこともない。
func deleteExportWithObject(
	ctx context.Context,
	objectStorage ExportObjectStorage,
	exportRepo *repository.ExportRepository,
	export *model.Export,
) error {
	objectKey := ExportObjectKey(export.ProfileID, export.ID)
	if err := objectStorage.Delete(ctx, objectKey); err != nil {
		return fmt.Errorf("エクスポートのオブジェクト削除に失敗 (export_id: %s, key: %s): %w", export.ID.String(), objectKey, err)
	}

	removed, err := exportRepo.Delete(ctx, export.ID)
	if err != nil {
		return fmt.Errorf("エクスポート行の削除に失敗 (export_id: %s): %w", export.ID.String(), err)
	}
	if !removed {
		// 行がすでに無い状態で、同じ削除の並行実行が残すのがこれである。どちらの
		// 実行も同じ最終状態を求めているため、これは処理の競合ではなく処理が済んだこと
		// を意味する。
		return nil
	}

	slog.InfoContext(ctx, "エクスポートを削除しました",
		"export_id", export.ID.String(),
		"profile_id", export.ProfileID.String(),
		"status", export.Status.String(),
	)
	return nil
}
