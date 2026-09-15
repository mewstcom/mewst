package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

const (
	// CleanupOrphanExportObjectsTimeoutは掃除1回の実行の上限。掃除は
	// エクスポートのプレフィックス配下の全オブジェクトを走査するため、コストは固定量
	// ではなく保存済みアーカイブ数に従う。この上限は、応答しなくなった一覧取得がworkerを
	// 無期限に占有することを防ぐ。
	CleanupOrphanExportObjectsTimeout = 10 * time.Minute

	// exportOrphanGraceは、掃除が削除を検討するまでにオブジェクトが変更されず
	// 経過している必要のある時間。一覧取得と照合は別の手順であり、アップロードはそれを
	// 記録する遷移より先に完了するため、書き込まれたばかりのオブジェクトには、まだそれを
	// 保持する行が無いことが正当にありうる。猶予時間はこれらの窓を、掃除が実際に手を
	// つけるオブジェクトから遠ざける。
	exportOrphanGrace = 24 * time.Hour

	// defaultExportOrphanBatchSizeは、1回のクエリでexportsテーブルと照合する
	// オブジェクト数。一覧取得はキーを1件ずつ渡すため走査のメモリはO(1) に保たれる。
	// バッチ化はその性質を保ったまま、クエリ回数をオブジェクト数より十分小さくする。
	defaultExportOrphanBatchSize = 100

	// defaultExportOrphanScanBudgetは、掃除1回が残りを継続ジョブへ引き渡すまでに
	// 走査するオブジェクト数。timeoutが許す範囲より十分小さくし、長い走査を終わらせるのが
	// timeoutではなく予算になるようにする。timeoutで止まった実行は同じ位置から再試行され
	// 同じ場所で失敗し続けるが、予算で止まった実行は次の実行が続きから始める位置を引き渡す
	// ためである。
	defaultExportOrphanScanBudget = 10_000
)

// errExportOrphanScanBudgetReachedは、実行が予算の分だけオブジェクトを走査した
// ところで走査を止める。一覧取得を途中で終わらせる手段は他に無いため、これはエラーとして
// 一覧取得を遡り、報告されるのではなく掃除自身に判別される。
var errExportOrphanScanBudgetReached = errors.New("孤児回収の走査予算に到達")

// ExportOrphanSweepLimitsは掃除1回の実行を有界にする。BatchSizeは1クエリで
// 照合する一覧済みオブジェクト数を、ScanBudgetは1回の実行が走査するオブジェクトの
// 総数を制限する。これにより、1回の実行に収まらないプレフィックスは、終端に到達しない
// 1回の実行ではなく、連なった複数回の実行で網羅される。
type ExportOrphanSweepLimits struct {
	BatchSize  int
	ScanBudget int
}

// DefaultExportOrphanSweepLimitsは本番で使う上限値を返す。正でない値はこれで
// 置き換えるため、呼び出し側の設定によって掃除が黙って何もしない状態にはならない。
func DefaultExportOrphanSweepLimits() ExportOrphanSweepLimits {
	return ExportOrphanSweepLimits{
		BatchSize:  defaultExportOrphanBatchSize,
		ScanBudget: defaultExportOrphanScanBudget,
	}
}

// normalizedは正でない値を既定値で置き換えた上限値を返す。
func (l ExportOrphanSweepLimits) normalized() ExportOrphanSweepLimits {
	defaults := DefaultExportOrphanSweepLimits()
	if l.BatchSize <= 0 {
		l.BatchSize = defaults.BatchSize
	}
	if l.ScanBudget <= 0 {
		l.ScanBudget = defaults.ScanBudget
	}
	return l
}

// CleanupOrphanExportObjectsUsecaseは、どのエクスポートからも保持されなくなった
// エクスポートオブジェクトを削除する。オブジェクトが孤児になるのは、エクスポートの
// 終端遷移とそのオブジェクトの削除の間でプロセスが終了した場合と、すでにアップロードを
// 終えていた停滞試行をリコンシリエーションが閉じた場合である。この掃除が無いと、
// これらのオブジェクトは永久に課金対象として残る。本機能が繰り返さないと決めた失敗が
// それである。
//
// 掃除はDBではなくオブジェクトストレージから始める。行の無いオブジェクトは行を
// 検索しても見つけられないため。一覧された各キーは自身のエクスポートを表すので、
// オブジェクトをまだ保持している行が、一覧されたどのオブジェクトが孤児かを答える。
//
// 1回の実行を有界にするのはプレフィックスの大きさではなく走査予算である。予算を
// 使い切った実行は、止まった位置のキーを持つ継続ジョブを投入するため、走査は実行を
// またいで前進する。位置は保存しない。処理されなかった継続ジョブは、プレフィックスの
// 残りを次回の定期的な掃除に委ねるだけで、その掃除はまた先頭から始まる。
type CleanupOrphanExportObjectsUsecase struct {
	exportRepo    *repository.ExportRepository
	objectStorage ExportObjectStorage
	dispatcher    *dispatcher.Dispatcher
	limits        ExportOrphanSweepLimits
}

// NewCleanupOrphanExportObjectsUsecaseは
// CleanupOrphanExportObjectsUsecaseを生成する。1回の実行を別の上限で区切る理由が
// なければDefaultExportOrphanSweepLimits() を渡す。
func NewCleanupOrphanExportObjectsUsecase(
	exportRepo *repository.ExportRepository,
	objectStorage ExportObjectStorage,
	d *dispatcher.Dispatcher,
	limits ExportOrphanSweepLimits,
) *CleanupOrphanExportObjectsUsecase {
	return &CleanupOrphanExportObjectsUsecase{
		exportRepo:    exportRepo,
		objectStorage: objectStorage,
		dispatcher:    d,
		limits:        limits.normalized(),
	}
}

// exportObjectCandidateは、検討対象になる程度に古い一覧済みオブジェクト1件と、
// そのキーが示すエクスポート。
type exportObjectCandidate struct {
	key      string
	exportID model.ExportID
}

// exportOrphanSweepCountsは1回の掃除の結果を集計する。削除の失敗はその場で
// 返さず数える。ストレージが拒否する1件のオブジェクトが、その後ろにあるオブジェクトへ
// 掃除が到達するのを止めてはならないためで、最後にそれらの失敗を1つのエラーとして
// 報告するのがこの集計である。
type exportOrphanSweepCounts struct {
	deleted int
	failed  int
	lastErr error
}

// ExecuteはstartAfterより後ろのエクスポートオブジェクトを一覧し、バッチ単位で
// exportsテーブルと照合して、どのエクスポートからも保持されていないものを削除する。
// startAfterが空ならプレフィックスを先頭から走査する。日次スケジュールが求めるのは
// これである。走査予算を使い切った実行は、残りのための継続ジョブを投入する。
func (uc *CleanupOrphanExportObjectsUsecase) Execute(ctx context.Context, startAfter string) error {
	cutoff := time.Now().Add(-exportOrphanGrace)

	var (
		batch    []exportObjectCandidate
		counts   exportOrphanSweepCounts
		matchErr error
		scanned  int
		lastKey  string
	)

	// flushはそこまでに溜まったバッチをexportsテーブルと照合し、その中の孤児を
	// 削除する。照合の失敗はmatchErrに記録する。この関数は一覧取得の内側から呼ばれる
	// ため、ここで返した失敗はListPrefixにラップされた形でExecuteの末尾に届き、
	// 一覧取得自体の失敗と区別できなくなる。両者は別のシステムを指すため、掃除はこれらを
	// 分けて保持し、照合の失敗は照合の失敗として報告する。
	flush := func(ctx context.Context) {
		if len(batch) == 0 {
			return
		}
		matchErr = uc.deleteOrphans(ctx, batch, &counts)
		batch = batch[:0]
	}

	listErr := uc.objectStorage.ListPrefix(ctx, ExportObjectKeyPrefix, startAfter, func(key string, lastModified time.Time) error {
		// 渡されたオブジェクトは、候補になるかどうかに関わらず数え、位置を進める。
		// 位置は継続ジョブが再開する場所であり、掃除が通り過ぎたオブジェクトはその後ろに
		// なければならないため。
		scanned++
		lastKey = key

		uc.collect(ctx, key, lastModified, cutoff, &batch)

		if len(batch) >= uc.limits.BatchSize {
			flush(ctx)
			if matchErr != nil {
				return matchErr
			}
		}
		if scanned >= uc.limits.ScanBudget {
			return errExportOrphanScanBudgetReached
		}
		return nil
	})
	budgetReached := errors.Is(listErr, errExportOrphanScanBudgetReached)

	// 一覧取得は埋まりきっていないバッチを残して終わり、バッチサイズの判定は
	// そこに到達しない。失敗した一覧取得は走査を途中で終えているため、その場合は
	// 完了させるものが無い。
	if listErr == nil || budgetReached {
		flush(ctx)
	}

	// 集計はエラーを返す前に記録する。失敗で終わる実行でも、そこへ至るまでに
	// 回収した内容が残るようにするため。
	slog.InfoContext(ctx, "孤児のエクスポートオブジェクトを回収しました",
		"deleted_count", counts.deleted,
		"failed_count", counts.failed,
		"scanned_count", scanned,
	)

	switch {
	case matchErr != nil:
		return matchErr
	case listErr != nil && !budgetReached:
		return fmt.Errorf("エクスポートオブジェクトの一覧取得に失敗: %w", listErr)
	case counts.failed > 0:
		return fmt.Errorf("孤児オブジェクトの削除に失敗した件数: %d (最後のエラー: %w)", counts.failed, counts.lastErr)
	}

	// 継続ジョブを投入するのは、その実行に報告すべきものが無くなってからにする。
	// 失敗した実行はジョブキューが同じ位置から再試行するため、ここで残りを引き渡すと
	// 再試行と継続ジョブが同じオブジェクトを走査することになる。
	if budgetReached {
		return uc.enqueueContinuation(ctx, lastKey)
	}
	return nil
}

// collectは、検討対象になる程度に古く、キーがエクスポートを示すオブジェクトを
// バッチへ追加する。
func (uc *CleanupOrphanExportObjectsUsecase) collect(
	ctx context.Context,
	key string,
	lastModified, cutoff time.Time,
	batch *[]exportObjectCandidate,
) {
	if lastModified.After(cutoff) {
		return
	}

	_, exportID, err := ParseExportObjectKey(key)
	if err != nil {
		// エクスポートのプレフィックス配下にあってキー規約に従わないオブジェクトは、
		// どのエクスポートのものか判別できないため手を付けない。説明できないものを削除
		// することは、残すことよりも悪い結果になるため。
		slog.WarnContext(ctx, "規約に従わないエクスポートオブジェクトを無視しました", "key", key, "error", err)
		return
	}

	*batch = append(*batch, exportObjectCandidate{key: key, exportID: exportID})
}

// enqueueContinuationは走査の残りを別のジョブへ引き渡し、この実行が止まったキーの
// 次から再開させる。位置は保存せずジョブの引数で運ぶため、掃除は実行と実行の間に自身の
// 状態を持たない。
func (uc *CleanupOrphanExportObjectsUsecase) enqueueContinuation(ctx context.Context, startAfter string) error {
	inserted, err := uc.dispatcher.EnqueueCleanupOrphanExportObjects(ctx, startAfter)
	if err != nil {
		return fmt.Errorf("孤児回収の継続ジョブの投入に失敗 (start_after: %s): %w", startAfter, err)
	}
	if inserted {
		slog.InfoContext(ctx, "孤児回収の継続ジョブを投入しました", "start_after", startAfter)
	}
	return nil
}

// deleteOrphansは、バッチのうちエクスポートがオブジェクトを保持しなくなった
// 候補を削除する。照合の失敗はエラーを返して掃除を終える。照合できないバッチは孤児と
// 区別できないためである。削除の失敗は数えたうえで先へ進む。
func (uc *CleanupOrphanExportObjectsUsecase) deleteOrphans(
	ctx context.Context,
	batch []exportObjectCandidate,
	counts *exportOrphanSweepCounts,
) error {
	ids := make([]model.ExportID, len(batch))
	for i, candidate := range batch {
		ids[i] = candidate.exportID
	}

	retainedIDs, err := uc.exportRepo.FindIDsRetainingObject(ctx, ids)
	if err != nil {
		return fmt.Errorf("オブジェクトを保持するエクスポートの照合に失敗: %w", err)
	}

	retained := make(map[model.ExportID]struct{}, len(retainedIDs))
	for _, id := range retainedIDs {
		retained[id] = struct{}{}
	}

	for _, candidate := range batch {
		if _, ok := retained[candidate.exportID]; ok {
			continue
		}
		if err := uc.objectStorage.Delete(ctx, candidate.key); err != nil {
			counts.failed++
			counts.lastErr = err
			slog.ErrorContext(ctx, "孤児のエクスポートオブジェクトの削除に失敗しました", "key", candidate.key, "error", err)
			continue
		}
		counts.deleted++
		slog.InfoContext(ctx, "孤児のエクスポートオブジェクトを削除しました", "key", candidate.key)
	}
	return nil
}
