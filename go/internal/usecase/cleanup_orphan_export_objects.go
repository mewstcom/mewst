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
	// CleanupOrphanExportObjectsTimeout bounds one sweep. The sweep walks every
	// object under the export prefix, so its cost follows the number of stored
	// archives rather than a fixed amount of work, and the bound keeps a listing
	// that stopped responding from occupying a worker indefinitely.
	//
	// [Ja] CleanupOrphanExportObjectsTimeout は掃除 1 回の実行の上限。掃除は
	// エクスポートのプレフィックス配下の全オブジェクトを走査するため、コストは固定量
	// ではなく保存済みアーカイブ数に従う。この上限は、応答しなくなった一覧取得が worker を
	// 無期限に占有することを防ぐ。
	CleanupOrphanExportObjectsTimeout = 10 * time.Minute

	// exportOrphanGrace is how long an object must have been untouched before the
	// sweep considers deleting it. Listing and matching are separate steps, and
	// an upload finishes before the transition that records it, so a recently
	// written object can legitimately have no row that retains it yet. The grace
	// keeps those windows far away from the objects the sweep acts on.
	//
	// [Ja] exportOrphanGrace は、掃除が削除を検討するまでにオブジェクトが変更されず
	// 経過している必要のある時間。一覧取得と照合は別の手順であり、アップロードはそれを
	// 記録する遷移より先に完了するため、書き込まれたばかりのオブジェクトには、まだそれを
	// 保持する行が無いことが正当にありうる。猶予時間はこれらの窓を、掃除が実際に手を
	// つけるオブジェクトから遠ざける。
	exportOrphanGrace = 24 * time.Hour

	// defaultExportOrphanBatchSize is how many listed objects are matched against
	// the exports table in one query. The listing hands over one key at a time so
	// the walk stays O(1) in memory; batching keeps that property while making the
	// number of queries a fraction of the number of objects.
	//
	// [Ja] defaultExportOrphanBatchSize は、1 回のクエリで exports テーブルと照合する
	// オブジェクト数。一覧取得はキーを 1 件ずつ渡すため走査のメモリは O(1) に保たれる。
	// バッチ化はその性質を保ったまま、クエリ回数をオブジェクト数より十分小さくする。
	defaultExportOrphanBatchSize = 100

	// defaultExportOrphanScanBudget is how many objects one sweep walks past
	// before handing the rest to a continuation job. It is set well inside what
	// the timeout allows so the budget, not the timeout, is what ends a long
	// walk: a run stopped by the timeout is retried from the same position and
	// would keep failing at the same place, while a run stopped by the budget
	// hands over a position the next run starts after.
	//
	// [Ja] defaultExportOrphanScanBudget は、掃除 1 回が残りを継続ジョブへ引き渡すまでに
	// 走査するオブジェクト数。timeout が許す範囲より十分小さくし、長い走査を終わらせるのが
	// timeout ではなく予算になるようにする。timeout で止まった実行は同じ位置から再試行され
	// 同じ場所で失敗し続けるが、予算で止まった実行は次の実行が続きから始める位置を引き渡す
	// ためである。
	defaultExportOrphanScanBudget = 10_000
)

// errExportOrphanScanBudgetReached stops the walk when the run has scanned as
// many objects as its budget allows. The listing has no other way to be ended
// early, so this travels back through it as an error and is recognized by the
// sweep rather than reported.
//
// [Ja] errExportOrphanScanBudgetReached は、実行が予算の分だけオブジェクトを走査した
// ところで走査を止める。一覧取得を途中で終わらせる手段は他に無いため、これはエラーとして
// 一覧取得を遡り、報告されるのではなく掃除自身に判別される。
var errExportOrphanScanBudgetReached = errors.New("孤児回収の走査予算に到達")

// ExportOrphanSweepLimits bounds one sweep. BatchSize limits how many listed
// objects are matched in one query, and ScanBudget limits how many objects the
// run walks past in total, so a prefix larger than one run is covered by a
// chain of runs instead of one run that never reaches the end.
//
// [Ja] ExportOrphanSweepLimits は掃除 1 回の実行を有界にする。BatchSize は 1 クエリで
// 照合する一覧済みオブジェクト数を、ScanBudget は 1 回の実行が走査するオブジェクトの
// 総数を制限する。これにより、1 回の実行に収まらないプレフィックスは、終端に到達しない
// 1 回の実行ではなく、連なった複数回の実行で網羅される。
type ExportOrphanSweepLimits struct {
	BatchSize  int
	ScanBudget int
}

// DefaultExportOrphanSweepLimits returns the limits used in production. Values
// that are not positive are replaced with these, so a caller cannot configure a
// sweep into silently doing nothing.
//
// [Ja] DefaultExportOrphanSweepLimits は本番で使う上限値を返す。正でない値はこれで
// 置き換えるため、呼び出し側の設定によって掃除が黙って何もしない状態にはならない。
func DefaultExportOrphanSweepLimits() ExportOrphanSweepLimits {
	return ExportOrphanSweepLimits{
		BatchSize:  defaultExportOrphanBatchSize,
		ScanBudget: defaultExportOrphanScanBudget,
	}
}

// normalized returns the limits with non-positive values replaced by the
// defaults.
//
// [Ja] normalized は正でない値を既定値で置き換えた上限値を返す。
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

// CleanupOrphanExportObjectsUsecase deletes export objects that no export
// retains any more. Objects become orphans when a process stops between the
// terminal transition of an export and the deletion of its object, and when
// reconciliation closes a stale attempt that had already uploaded one. Without
// this sweep those objects would be billed forever, which is the failure this
// feature set out not to repeat.
//
// The sweep starts from the object storage rather than the database, because
// an object with no row cannot be found by querying rows. Each listed key names
// its export, so the rows that still retain their object answer which of the
// listed objects are orphans.
//
// One run is bounded by its scan budget rather than by the size of the prefix.
// A run that spends its budget enqueues a continuation carrying the key it
// stopped at, so the walk advances across runs. Nothing about the position is
// stored: a continuation that is never worked leaves the rest of the prefix to
// the next scheduled sweep, which starts from the beginning again.
//
// [Ja] CleanupOrphanExportObjectsUsecase は、どのエクスポートからも保持されなくなった
// エクスポートオブジェクトを削除する。オブジェクトが孤児になるのは、エクスポートの
// 終端遷移とそのオブジェクトの削除の間でプロセスが終了した場合と、すでにアップロードを
// 終えていた停滞試行をリコンシリエーションが閉じた場合である。この掃除が無いと、
// これらのオブジェクトは永久に課金対象として残る。本機能が繰り返さないと決めた失敗が
// それである。
//
// 掃除は DB ではなくオブジェクトストレージから始める。行の無いオブジェクトは行を
// 検索しても見つけられないため。一覧された各キーは自身のエクスポートを表すので、
// オブジェクトをまだ保持している行が、一覧されたどのオブジェクトが孤児かを答える。
//
// 1 回の実行を有界にするのはプレフィックスの大きさではなく走査予算である。予算を
// 使い切った実行は、止まった位置のキーを持つ継続ジョブを投入するため、走査は実行を
// またいで前進する。位置は保存しない。処理されなかった継続ジョブは、プレフィックスの
// 残りを次回の定期的な掃除に委ねるだけで、その掃除はまた先頭から始まる。
type CleanupOrphanExportObjectsUsecase struct {
	exportRepo    *repository.ExportRepository
	objectStorage ExportObjectStorage
	dispatcher    *dispatcher.Dispatcher
	limits        ExportOrphanSweepLimits
}

// NewCleanupOrphanExportObjectsUsecase creates a
// CleanupOrphanExportObjectsUsecase. Pass DefaultExportOrphanSweepLimits()
// unless the caller has a reason to bound a run differently.
//
// [Ja] NewCleanupOrphanExportObjectsUsecase は
// CleanupOrphanExportObjectsUsecase を生成する。1 回の実行を別の上限で区切る理由が
// なければ DefaultExportOrphanSweepLimits() を渡す。
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

// exportObjectCandidate is one listed object old enough to be considered,
// together with the export its key names.
//
// [Ja] exportObjectCandidate は、検討対象になる程度に古い一覧済みオブジェクト 1 件と、
// そのキーが示すエクスポート。
type exportObjectCandidate struct {
	key      string
	exportID model.ExportID
}

// exportOrphanSweepCounts accumulates what one sweep did. Deletion failures are
// counted rather than returned right away: one object the storage refuses must
// not stop the sweep from reaching the objects behind it, and the counts are
// what turns those failures into a single reported error at the end.
//
// [Ja] exportOrphanSweepCounts は 1 回の掃除の結果を集計する。削除の失敗はその場で
// 返さず数える。ストレージが拒否する 1 件のオブジェクトが、その後ろにあるオブジェクトへ
// 掃除が到達するのを止めてはならないためで、最後にそれらの失敗を 1 つのエラーとして
// 報告するのがこの集計である。
type exportOrphanSweepCounts struct {
	deleted int
	failed  int
	lastErr error
}

// Execute lists the export objects after startAfter, matches them against the
// exports table in batches and deletes the ones no export retains. An empty
// startAfter walks the prefix from the beginning, which is what the daily
// schedule asks for; a run that spends its scan budget enqueues a continuation
// for the rest.
//
// [Ja] Execute は startAfter より後ろのエクスポートオブジェクトを一覧し、バッチ単位で
// exports テーブルと照合して、どのエクスポートからも保持されていないものを削除する。
// startAfter が空ならプレフィックスを先頭から走査する。日次スケジュールが求めるのは
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

	// flush matches the batch accumulated so far against the exports table and
	// deletes the orphans in it, recording a failed match in matchErr. It is
	// called from inside the listing, so a failure returned from here reaches the
	// end of Execute wrapped by ListPrefix, where it can no longer be told apart
	// from a failure of the listing itself. The two name different systems, so
	// the sweep keeps them separate and reports the match failure as itself.
	//
	// [Ja] flush はそこまでに溜まったバッチを exports テーブルと照合し、その中の孤児を
	// 削除する。照合の失敗は matchErr に記録する。この関数は一覧取得の内側から呼ばれる
	// ため、ここで返した失敗は ListPrefix にラップされた形で Execute の末尾に届き、
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
		// Every object handed over counts and moves the position, whether or not
		// it becomes a candidate. The position is where a continuation resumes, so
		// an object the sweep walked past has to be behind it.
		//
		// [Ja] 渡されたオブジェクトは、候補になるかどうかに関わらず数え、位置を進める。
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

	// The listing ends on a partly filled batch, which the batch-size check never
	// reaches. A listing that failed leaves the walk in the middle, so there is
	// nothing to complete then.
	//
	// [Ja] 一覧取得は埋まりきっていないバッチを残して終わり、バッチサイズの判定は
	// そこに到達しない。失敗した一覧取得は走査を途中で終えているため、その場合は
	// 完了させるものが無い。
	if listErr == nil || budgetReached {
		flush(ctx)
	}

	// The counts are logged before any error is returned, so a run that ends on a
	// failure still records what it collected before reaching it.
	//
	// [Ja] 集計はエラーを返す前に記録する。失敗で終わる実行でも、そこへ至るまでに
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

	// The continuation is enqueued only once the run has nothing to report. A run
	// that failed is retried by the job queue from the same position, so handing
	// the rest over now would have the retry and the continuation walk the same
	// objects.
	//
	// [Ja] 継続ジョブを投入するのは、その実行に報告すべきものが無くなってからにする。
	// 失敗した実行はジョブキューが同じ位置から再試行するため、ここで残りを引き渡すと
	// 再試行と継続ジョブが同じオブジェクトを走査することになる。
	if budgetReached {
		return uc.enqueueContinuation(ctx, lastKey)
	}
	return nil
}

// collect adds the object to the batch when it is old enough to be considered
// and its key names an export.
//
// [Ja] collect は、検討対象になる程度に古く、キーがエクスポートを示すオブジェクトを
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
		// An object under the export prefix that does not follow the key
		// convention cannot be attributed to an export, so the sweep leaves it
		// alone: deleting what it cannot explain is the one outcome worse than
		// keeping it.
		//
		// [Ja] エクスポートのプレフィックス配下にあってキー規約に従わないオブジェクトは、
		// どのエクスポートのものか判別できないため手を付けない。説明できないものを削除
		// することは、残すことよりも悪い結果になるため。
		slog.WarnContext(ctx, "規約に従わないエクスポートオブジェクトを無視しました", "key", key, "error", err)
		return
	}

	*batch = append(*batch, exportObjectCandidate{key: key, exportID: exportID})
}

// enqueueContinuation hands the rest of the walk to another job, resuming after
// the key this run stopped at. The position travels in the job's arguments
// rather than being stored, so the sweep keeps no state of its own between
// runs.
//
// [Ja] enqueueContinuation は走査の残りを別のジョブへ引き渡し、この実行が止まったキーの
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

// deleteOrphans deletes every candidate in the batch whose export no longer
// retains its object. A failed lookup returns an error and ends the sweep,
// because without it the batch cannot be told apart from orphans; a failed
// deletion is counted and the sweep moves on.
//
// [Ja] deleteOrphans は、バッチのうちエクスポートがオブジェクトを保持しなくなった
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
