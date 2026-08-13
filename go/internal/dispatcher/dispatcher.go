// Package dispatcher はジョブキューへの投入を抽象化する。
// Repository がデータベースアクセスを抽象化するのと同じ発想で、
// Dispatcher がジョブキューアクセスを抽象化する。
package dispatcher

import (
	"context"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// QueueExport is the River queue that runs export generation. Generation is
// long-running and streams a whole archive, so it is kept out of the default
// queue: a slow or stuck generation must not occupy the workers that deliver
// timelines and send emails.
//
// [Ja] QueueExport はエクスポート生成を実行する River のキュー。生成は長時間かかり
// アーカイブ全体をストリーミングするため、既定キューから分離する。遅い生成や停止した
// 生成が、タイムライン配信やメール送信の worker を占有しないようにするため。
const QueueExport = "export"

// --- ジョブ引数型 ---

// SendEmailConfirmationArgs はメール確認コード送信ジョブの引数
type SendEmailConfirmationArgs struct {
	Email  string `json:"email"`
	Code   string `json:"code"`
	Locale string `json:"locale"`
}

// Kind はジョブの種類を返す
func (SendEmailConfirmationArgs) Kind() string { return "send_email_confirmation" }

// InsertOpts returns the job's Insert options. Priority is demoted to 2 (below
// the default 1) so the timeline delivery jobs (fanout_post /
// add_post_to_timeline), which keep the default top priority, are fetched first
// when the worker pool is saturated. Mirrors the Rails :high priority queue used
// for those jobs.
//
// [Ja] InsertOpts はジョブの Insert オプションを返す。Priority を 2 (既定の 1 より
// 低く) に降格し、worker プールが飽和したときに、既定の最高優先度を保つタイムライン
// 配信ジョブ (fanout_post / add_post_to_timeline) を先に fetch させる。Rails で
// これらのジョブが :high 優先度キューを使うのに合わせる。
func (SendEmailConfirmationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5, Priority: 2}
}

// FanoutPostArgs is the argument for the job that fans a published post out to
// its author's followers' home timelines.
// [Ja] FanoutPostArgs は公開された投稿を投稿者のフォロワーのホームタイムラインへ
// 配信するジョブの引数。
type FanoutPostArgs struct {
	PostID string `json:"post_id"`
}

// Kind はジョブの種類を返す
func (FanoutPostArgs) Kind() string { return "fanout_post" }

// InsertOpts returns the job's Insert options. Priority stays at the default
// (1, the highest) so timeline fanout is worked before lower-priority jobs such
// as confirmation emails (Priority 2).
//
// [Ja] InsertOpts はジョブの Insert オプションを返す。Priority は既定 (1, 最高) の
// ままにし、確認メール (Priority 2) など優先度の低いジョブより先に fanout を処理
// させる。
func (FanoutPostArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// AddPostToTimelineArgs is the argument for the job that adds a single post to a
// single profile's home timeline (one job per follower, enqueued by fanout).
// [Ja] AddPostToTimelineArgs は 1 件の投稿を 1 つのプロフィールのホームタイムラインに
// 追加するジョブの引数 (フォロワー 1 人につき 1 ジョブを fanout が enqueue する)。
type AddPostToTimelineArgs struct {
	ProfileID string `json:"profile_id"`
	PostID    string `json:"post_id"`
}

// Kind はジョブの種類を返す
func (AddPostToTimelineArgs) Kind() string { return "add_post_to_timeline" }

// InsertOpts returns the job's Insert options. Priority stays at the default
// (1, the highest) so timeline delivery is worked before lower-priority jobs
// such as confirmation emails (Priority 2).
//
// [Ja] InsertOpts はジョブの Insert オプションを返す。Priority は既定 (1, 最高) の
// ままにし、確認メール (Priority 2) など優先度の低いジョブより先にタイムライン配信を
// 処理させる。
func (AddPostToTimelineArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// exportJobUniqueStates returns the job states across which export jobs are
// unique: River's default set without completed. Uniqueness here has to mean
// "a job for this work intent is still waiting or running", not "a job for this
// work intent ran recently". Every job using this helper is re-derived from durable
// database state (a queued row, a success with no notification, exports older
// than the latest success) and re-inserted by reconciliation, so a job that
// completed without converging its row must not block the next insert until the
// job cleaner removes it.
//
// River documents retryable as the only state that can safely be removed from
// the default set. Dropping completed still works for the insert path, because
// the partial unique index only covers rows whose current state is in the set.
// The cost is that retrying a completed export job by hand puts the row back
// into available without any uniqueness check, so that retry fails on the
// unique index whenever a newer job for the same work intent is already waiting
// or running.
//
// [Ja] exportJobUniqueStates はエクスポート系ジョブの一意性を判定する状態集合を返す
// (River の既定集合から completed を除いたもの)。ここでの一意性は「この作業依頼の
// ジョブがまだ待機中または実行中」を意味する必要があり、「最近実行された」ではない。
// このヘルパーを使うジョブは DB の永続状態 (queued 行、未通知の成功、最新成功より
// 古いエクスポート) からリコンシリエーションが再導出して再投入するため、行を収束
// させないまま完了したジョブが、job cleaner に消されるまで次の投入を塞いではならない。
//
// River が既定集合から安全に外せる状態として挙げているのは retryable だけである。
// completed を外しても投入経路は正しく動く。部分ユニークインデックスは、現在の状態が
// 集合に含まれる行だけを対象とするため。代償として、完了したエクスポート系ジョブを
// 手動で再試行すると、一意性を判定しないまま行が available へ戻るため、同じ作業依頼の
// 新しいジョブが既に待機中または実行中であれば、その再試行がユニークインデックスで
// 失敗する。
func exportJobUniqueStates() []rivertype.JobState {
	return []rivertype.JobState{
		rivertype.JobStateAvailable,
		rivertype.JobStatePending,
		rivertype.JobStateRetryable,
		rivertype.JobStateRunning,
		rivertype.JobStateScheduled,
	}
}

// GenerateExportMaxAttempts is how many times the job queue runs a generation
// job for one export before the export is given up on. It is exported because
// reconciliation needs the same number: an export left started by a process
// that died is only closed as failed once its attempts are used up, and a
// number that drifted from this one would either give up early or retry a
// hopeless export forever.
//
// [Ja] GenerateExportMaxAttempts は、1 件のエクスポートを諦めるまでにジョブキューが
// 生成ジョブを実行する回数。リコンシリエーションが同じ回数を必要とするため exported
// にしている。プロセスの異常終了で started のまま残ったエクスポートは、試行を使い
// 切って初めて failed として閉じられるため、この値とずれた回数を持つと、早すぎる
// 打ち切りか、見込みのないエクスポートの無限の再試行のどちらかになる。
const GenerateExportMaxAttempts = 5

// GenerateExportArgs is the argument for the job that generates one export's
// zip and streams it to the object storage.
//
// [Ja] GenerateExportArgs は 1 件のエクスポートの zip を生成し、オブジェクト
// ストレージへストリーミングするジョブの引数。
type GenerateExportArgs struct {
	ExportID string `json:"export_id"`
}

// Kind returns the job's type.
//
// [Ja] Kind はジョブの種類を返す。
func (GenerateExportArgs) Kind() string { return "generate_export" }

// InsertOpts returns the job's Insert options. Generation runs on the dedicated
// export queue and retries up to five times so that a transient database or
// object storage failure does not end an export. Uniqueness is per export ID so
// that the immediate insert made after Create commits and a later
// reconciliation insert cannot generate the same export twice.
//
// [Ja] InsertOpts はジョブの Insert オプションを返す。生成は専用の export キューで
// 実行し、DB やオブジェクトストレージの一時的な失敗でエクスポートを終わらせないよう
// 最大 5 回まで再試行する。一意性はエクスポート ID 単位とし、Create のコミット後の
// 即時投入とその後のリコンシリエーションの投入で同じエクスポートを二重に生成しない
// ようにする。
func (GenerateExportArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueExport,
		MaxAttempts: GenerateExportMaxAttempts,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: exportJobUniqueStates(),
		},
	}
}

// CleanupOldExportsArgs is the argument for the job that deletes one profile's
// exports older than its latest success, object first and row second.
//
// [Ja] CleanupOldExportsArgs は 1 つのプロフィールについて、最新の成功より古い
// エクスポートをオブジェクト → 行の順に削除するジョブの引数。
type CleanupOldExportsArgs struct {
	ProfileID string `json:"profile_id"`
}

// Kind returns the job's type.
//
// [Ja] Kind はジョブの種類を返す。
func (CleanupOldExportsArgs) Kind() string { return "cleanup_old_exports" }

// InsertOpts returns the job's Insert options. Cleanup is demoted to Priority 3
// (below the confirmation email at 2 and timeline delivery at the default 1)
// because nobody waits on it: the new export is already downloadable, and the
// old one is already excluded from the download. Uniqueness is per profile so
// that the insert made after a success and the reconciliation insert collapse
// into one run.
//
// [Ja] InsertOpts はジョブの Insert オプションを返す。削除は Priority 3 (確認メールの
// 2、タイムライン配信の既定 1 より低い) に降格する。新しいエクスポートは既に
// ダウンロード可能で、古いものは既にダウンロード対象から外れており、誰も待っていない
// ため。一意性はプロフィール単位とし、成功後の投入とリコンシリエーションの投入を
// 1 回の実行にまとめる。
func (CleanupOldExportsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 5,
		Priority:    3,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: exportJobUniqueStates(),
		},
	}
}

// SendExportCompletedEmailArgs is the argument for the job that notifies the
// requester that their export is ready to download.
//
// [Ja] SendExportCompletedEmailArgs はエクスポートがダウンロード可能になったことを
// 申請者へ通知するジョブの引数。
type SendExportCompletedEmailArgs struct {
	ExportID string `json:"export_id"`
}

// Kind returns the job's type.
//
// [Ja] Kind はジョブの種類を返す。
func (SendExportCompletedEmailArgs) Kind() string { return "send_export_completed_email" }

// InsertOpts returns the job's Insert options. Priority matches the
// confirmation email (2) because both are transactional mail a user is waiting
// for. Uniqueness is per export so that the insert made after a success and the
// reconciliation insert for an export with no notification collapse into one
// send.
//
// [Ja] InsertOpts はジョブの Insert オプションを返す。Priority は確認メールと同じ
// 2 とする。どちらもユーザーが待っている transactional なメールであるため。
// 一意性はエクスポート単位とし、成功後の投入と未通知エクスポートに対する
// リコンシリエーションの投入を 1 回の送信にまとめる。
func (SendExportCompletedEmailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 5,
		Priority:    2,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: exportJobUniqueStates(),
		},
	}
}

// ReconcileExportsArgs is the argument for the periodic job that converges
// exports left behind by a failed insert or a process that stopped mid-flight.
// It carries no fields: the outstanding work is derived from the exports table
// at run time, never from the job arguments.
//
// [Ja] ReconcileExportsArgs は、投入の失敗や処理途中でのプロセス終了によって
// 取り残されたエクスポートを収束させる定期ジョブの引数。フィールドを持たない。
// 未処理の作業はジョブ引数からではなく、実行時に exports テーブルから導出する。
type ReconcileExportsArgs struct{}

// Kind returns the job's type.
//
// [Ja] Kind はジョブの種類を返す。
func (ReconcileExportsArgs) Kind() string { return "reconcile_exports" }

// InsertOpts returns the job's Insert options. MaxAttempts is 1 because the
// next periodic run is the retry: retrying in place would keep a failed job
// retryable, and a retryable job blocks the next periodic insert through the
// uniqueness check. Priority is demoted to 3 for the same reason as cleanup —
// recovery is maintenance, and no request is waiting on it.
//
// [Ja] InsertOpts はジョブの Insert オプションを返す。次回の定期実行が再試行に
// あたるため MaxAttempts は 1 とする。その場で再試行すると失敗したジョブが
// retryable のまま残り、一意性判定によって次の定期投入を塞いでしまう。Priority は
// cleanup と同じ理由で 3 に降格する (回復処理は保守作業であり、待っているリクエストは
// ない)。
func (ReconcileExportsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 1,
		Priority:    3,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: exportJobUniqueStates(),
		},
	}
}

// CleanupOrphanExportObjectsPeriod is both the orphan sweep's periodic
// interval and its uniqueness window. Sharing the value keeps a leader's
// run-on-start insert and the regular schedule in the same daily contract:
// once a sweep completes in a window, another leader can attempt the insert
// without listing the export prefix again.
//
// [Ja] CleanupOrphanExportObjectsPeriod は孤児回収の定期実行間隔と一意性の時間枠を
// 兼ねる。値を共有することで、リーダーの起動時投入と通常スケジュールを同じ日次契約に
// 揃える。ある時間枠で掃除が完了した後は、別のリーダーが投入を試みてもエクスポートの
// プレフィックスを再度一覧しない。
const CleanupOrphanExportObjectsPeriod = 24 * time.Hour

// CleanupOrphanExportObjectsArgs is the argument for the periodic job that
// deletes export objects no export retains any more. It carries only where to
// resume the walk: which of the listed objects are orphans is still decided at
// run time by matching the object storage against the exports table, never
// taken from the job arguments.
//
// [Ja] CleanupOrphanExportObjectsArgs は、どのエクスポートからも保持されなくなった
// エクスポートオブジェクトを削除する定期ジョブの引数。持つのは走査を再開する位置だけ
// である。一覧されたどのオブジェクトが孤児かは、ジョブ引数からではなく、実行時に
// オブジェクトストレージと exports テーブルを照合することで求める。
type CleanupOrphanExportObjectsArgs struct {
	// StartAfter is the object key the walk resumes after. The daily schedule
	// inserts it empty so the walk starts at the beginning of the prefix; a run
	// that spends its scan budget inserts a continuation carrying the key it
	// stopped at, which is what lets a prefix larger than one run's budget be
	// covered by a chain of runs.
	//
	// [Ja] StartAfter は走査を再開する位置となるオブジェクトキー。日次スケジュールは
	// これを空で投入し、走査はプレフィックスの先頭から始まる。走査予算を使い切った実行
	// は止まったキーを持つ継続ジョブを投入する。1 回の実行の予算に収まらないプレフィックス
	// を、連なった複数回の実行で網羅できるのはこのためである。
	StartAfter string `json:"start_after,omitempty"`
}

// Kind returns the job's type.
//
// [Ja] Kind はジョブの種類を返す。
func (CleanupOrphanExportObjectsArgs) Kind() string { return "cleanup_orphan_export_objects" }

// InsertOpts returns the job's Insert options. Unlike reconcile_exports, the
// sweep retries in place: its schedule is daily, so leaving a transient object
// storage failure to the next run would keep orphans billed for another day,
// and River's backoff uses up the attempts in minutes. Its unique period is the
// daily schedule interval, and its states include completed, so every elected
// leader can insert it on start without repeating a successful listing in the
// same window. Priority is demoted to 3 for the same reason as the other
// maintenance jobs.
//
// Uniqueness is by args, so the daily window applies per resume position. The
// scheduled insert always carries the empty one, which is what collapses
// repeated run-on-start inserts of the first segment; a continuation carries a
// key of its own and is not suppressed by an earlier segment that completed in
// the same window, while a continuation from the same key is, which keeps a
// retried run from splitting the walk in two.
//
// [Ja] InsertOpts はジョブの Insert オプションを返す。reconcile_exports とは異なり、
// この掃除はその場で再試行する。スケジュールが日次のため、オブジェクトストレージの
// 一時的な失敗を次回の実行に委ねると孤児オブジェクトの課金がもう 1 日続くこと、および
// River のバックオフでは試行が数分で尽きることによる。一意性の時間枠は日次の実行間隔
// と揃え、状態集合には completed も含める。これにより、選出された各リーダーが起動時に
// 投入しても、同じ時間枠で成功済みの一覧は繰り返さない。Priority は他の保守ジョブと
// 同じ理由で 3 に降格する。
//
// 一意性は引数単位のため、日次の時間枠は再開位置ごとに効く。スケジュールからの投入は
// 常に空の位置を持ち、最初の区間に対する起動時投入の繰り返しをまとめるのはこれである。
// 継続ジョブは自身のキーを持つため、同じ時間枠で完了した手前の区間によって抑止されない。
// 一方、同じキーからの継続ジョブは抑止されるため、再試行された実行が走査を二股に
// 分けることはない。
func (CleanupOrphanExportObjectsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 5,
		Priority:    3,
		UniqueOpts: river.UniqueOpts{
			ByArgs:   true,
			ByPeriod: CleanupOrphanExportObjectsPeriod,
			ByState:  rivertype.UniqueOptsByStateDefault(),
		},
	}
}

// --- Dispatcher ---

// JobInserter はジョブをキューに追加するインターフェース
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Dispatcher はジョブキューへの投入を抽象化する
type Dispatcher struct {
	client JobInserter
}

// NewDispatcher は新しい Dispatcher を生成する
func NewDispatcher(client JobInserter) *Dispatcher {
	return &Dispatcher{client: client}
}

// DeferredInserter is a JobInserter whose backing inserter is wired after
// construction. It breaks the initialization cycle between the Dispatcher and
// the River client: a UseCase that enqueues jobs (e.g. fanout) needs a
// Dispatcher, the Dispatcher needs a JobInserter, but the River client that
// satisfies JobInserter can only be built after the Workers — which wrap those
// very UseCases — are registered. Build a Dispatcher around a DeferredInserter
// first, then call SetInserter once the River client exists.
//
// [Ja] DeferredInserter は実体の inserter を構築後に注入する JobInserter。
// Dispatcher と River クライアントの初期化循環を断つ: ジョブを enqueue する
// UseCase (fanout 等) は Dispatcher を必要とし、Dispatcher は JobInserter を必要と
// するが、JobInserter を満たす River クライアントは、その UseCase を内包する Worker を
// 登録した後でないと生成できない。先に DeferredInserter を包んだ Dispatcher を作り、
// River クライアント生成後に SetInserter を呼ぶ。
type DeferredInserter struct {
	inserter JobInserter
}

// SetInserter wires the backing inserter. Call it once, after the River client
// is created and before any job runs.
// [Ja] SetInserter は実体の inserter を注入する。River クライアント生成後・
// ジョブ実行前に一度だけ呼ぶ。
func (d *DeferredInserter) SetInserter(inserter JobInserter) {
	d.inserter = inserter
}

// Insert delegates to the wired inserter.
// [Ja] Insert は注入済みの inserter に委譲する。
func (d *DeferredInserter) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return d.inserter.Insert(ctx, args, opts)
}

// EnqueueEmailConfirmation はメール確認コード送信ジョブをキューに追加する
func (d *Dispatcher) EnqueueEmailConfirmation(ctx context.Context, email, code, locale string) error {
	args := SendEmailConfirmationArgs{Email: email, Code: code, Locale: locale}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueueFanoutPost enqueues the job that fans a post out to its author's
// followers. The post ID is taken as a string so that the dispatcher does not
// depend on the model package (callers pass postID.String()).
//
// [Ja] EnqueueFanoutPost は投稿を投稿者のフォロワーへ配信するジョブを enqueue する。
// dispatcher が model パッケージに依存しないよう post ID は文字列で受け取る
// (呼び出し側が postID.String() を渡す)。
func (d *Dispatcher) EnqueueFanoutPost(ctx context.Context, postID string) error {
	args := FanoutPostArgs{PostID: postID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueueAddPostToTimeline enqueues the job that adds a post to one profile's
// home timeline. The IDs are taken as strings to keep the dispatcher free of a
// model dependency (callers pass profileID.String() / postID.String()).
//
// [Ja] EnqueueAddPostToTimeline は投稿を 1 つのプロフィールのホームタイムラインに
// 追加するジョブを enqueue する。dispatcher を model 非依存に保つため ID は文字列で
// 受け取る (呼び出し側が profileID.String() / postID.String() を渡す)。
func (d *Dispatcher) EnqueueAddPostToTimeline(ctx context.Context, profileID, postID string) error {
	args := AddPostToTimelineArgs{ProfileID: profileID, PostID: postID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueueGenerateExport enqueues the job that generates an export's zip. The
// export ID is taken as a string to keep the dispatcher free of a model
// dependency (callers pass exportID.String()).
//
// It reports whether a job was actually inserted: River answers an insert made
// while a job for the same export is still outstanding with a result flagged as
// skipped rather than with an error, and reconciliation must not count a
// skipped candidate towards the number of candidates it takes on in one run.
//
// [Ja] EnqueueGenerateExport はエクスポートの zip を生成するジョブを enqueue する。
// dispatcher を model 非依存に保つため ID は文字列で受け取る (呼び出し側が
// exportID.String() を渡す)。
//
// 実際に投入したかどうかを返す。同じエクスポートのジョブが未完了のまま残っている
// ときの投入に対して、River はエラーではなく skip 済みの結果を返すため、
// リコンシリエーションは skip された候補を 1 回の実行で引き受けた件数に数えては
// ならない。
func (d *Dispatcher) EnqueueGenerateExport(ctx context.Context, exportID string) (bool, error) {
	args := GenerateExportArgs{ExportID: exportID}
	opts := args.InsertOpts()
	res, err := d.client.Insert(ctx, args, &opts)
	if err != nil {
		return false, err
	}
	return !res.UniqueSkippedAsDuplicate, nil
}

// EnqueueCleanupOldExports enqueues the job that deletes a profile's exports
// older than its latest success. The profile ID is taken as a string to keep
// the dispatcher free of a model dependency (callers pass profileID.String()).
//
// It reports whether a job was actually inserted, on the same terms as
// EnqueueGenerateExport.
//
// [Ja] EnqueueCleanupOldExports は、最新の成功より古いプロフィールのエクスポートを
// 削除するジョブを enqueue する。dispatcher を model 非依存に保つため ID は文字列で
// 受け取る (呼び出し側が profileID.String() を渡す)。
//
// EnqueueGenerateExport と同じ条件で、実際に投入したかどうかを返す。
func (d *Dispatcher) EnqueueCleanupOldExports(ctx context.Context, profileID string) (bool, error) {
	args := CleanupOldExportsArgs{ProfileID: profileID}
	opts := args.InsertOpts()
	res, err := d.client.Insert(ctx, args, &opts)
	if err != nil {
		return false, err
	}
	return !res.UniqueSkippedAsDuplicate, nil
}

// EnqueueSendExportCompletedEmail enqueues the job that notifies the requester
// that their export is ready. The export ID is taken as a string to keep the
// dispatcher free of a model dependency (callers pass exportID.String()).
//
// It reports whether a job was actually inserted, on the same terms as
// EnqueueGenerateExport.
//
// [Ja] EnqueueSendExportCompletedEmail はエクスポートの完成を申請者へ通知する
// ジョブを enqueue する。dispatcher を model 非依存に保つため ID は文字列で受け取る
// (呼び出し側が exportID.String() を渡す)。
//
// EnqueueGenerateExport と同じ条件で、実際に投入したかどうかを返す。
func (d *Dispatcher) EnqueueSendExportCompletedEmail(ctx context.Context, exportID string) (bool, error) {
	args := SendExportCompletedEmailArgs{ExportID: exportID}
	opts := args.InsertOpts()
	res, err := d.client.Insert(ctx, args, &opts)
	if err != nil {
		return false, err
	}
	return !res.UniqueSkippedAsDuplicate, nil
}

// EnqueueReconcileExports enqueues the job that converges exports left behind
// by a failed insert or a process that stopped mid-flight.
//
// It reports whether a job was actually inserted, on the same terms as
// EnqueueGenerateExport.
//
// [Ja] EnqueueReconcileExports は、投入の失敗や処理途中でのプロセス終了によって
// 取り残されたエクスポートを収束させるジョブを enqueue する。
//
// EnqueueGenerateExport と同じ条件で、実際に投入したかどうかを返す。
func (d *Dispatcher) EnqueueReconcileExports(ctx context.Context) (bool, error) {
	args := ReconcileExportsArgs{}
	opts := args.InsertOpts()
	res, err := d.client.Insert(ctx, args, &opts)
	if err != nil {
		return false, err
	}
	return !res.UniqueSkippedAsDuplicate, nil
}

// EnqueueCleanupOrphanExportObjects enqueues the sweep that deletes export
// objects no export retains any more, resuming after startAfter. An empty
// startAfter walks the prefix from the beginning; the sweep passes the key it
// stopped at to hand the rest of the walk over.
//
// It reports whether a job was actually inserted, on the same terms as
// EnqueueGenerateExport.
//
// [Ja] EnqueueCleanupOrphanExportObjects は、どのエクスポートからも保持されなくなった
// エクスポートオブジェクトを削除する掃除を、startAfter の次から再開する形で enqueue
// する。startAfter が空ならプレフィックスを先頭から走査する。掃除は走査の残りを
// 引き渡すために、自身が止まったキーを渡す。
//
// EnqueueGenerateExport と同じ条件で、実際に投入したかどうかを返す。
func (d *Dispatcher) EnqueueCleanupOrphanExportObjects(ctx context.Context, startAfter string) (bool, error) {
	args := CleanupOrphanExportObjectsArgs{StartAfter: startAfter}
	opts := args.InsertOpts()
	res, err := d.client.Insert(ctx, args, &opts)
	if err != nil {
		return false, err
	}
	return !res.UniqueSkippedAsDuplicate, nil
}
