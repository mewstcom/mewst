// Package dispatcherはジョブキューへの投入を抽象化する。
// Repositoryがデータベースアクセスを抽象化するのと同じ発想で、
// Dispatcherがジョブキューアクセスを抽象化する。
package dispatcher

import (
	"context"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// QueueExportはエクスポート生成を実行するRiverのキュー。生成は長時間かかり
// アーカイブ全体をストリーミングするため、既定キューから分離する。遅い生成や停止した
// 生成が、タイムライン配信やメール送信のworkerを占有しないようにするため。
const QueueExport = "export"

// --- ジョブ引数型 ---

// SendEmailConfirmationArgsはメール確認コード送信ジョブの引数
type SendEmailConfirmationArgs struct {
	Email  string `json:"email"`
	Code   string `json:"code"`
	Locale string `json:"locale"`
}

// Kindはジョブの種類を返す
func (SendEmailConfirmationArgs) Kind() string { return "send_email_confirmation" }

// InsertOptsはジョブのInsertオプションを返す。Priorityを2 (既定の1より
// 低く) に降格し、workerプールが飽和したときに、既定の最高優先度を保つタイムライン
// 配信ジョブ (fanout_post / add_post_to_timeline) を先にfetchさせる。Railsで
// これらのジョブが :high優先度キューを使うのに合わせる。
func (SendEmailConfirmationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5, Priority: 2}
}

// FanoutPostArgsは公開された投稿を投稿者のフォロワーのホームタイムラインへ
// 配信するジョブの引数。
type FanoutPostArgs struct {
	PostID string `json:"post_id"`
}

// Kindはジョブの種類を返す
func (FanoutPostArgs) Kind() string { return "fanout_post" }

// InsertOptsはジョブのInsertオプションを返す。Priorityは既定 (1, 最高) の
// ままにし、確認メール (Priority 2) など優先度の低いジョブより先にfanoutを処理
// させる。
func (FanoutPostArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// AddPostToTimelineArgsは1件の投稿を1つのプロフィールのホームタイムラインに
// 追加するジョブの引数 (フォロワー1人につき1ジョブをfanoutがenqueueする)。
type AddPostToTimelineArgs struct {
	ProfileID string `json:"profile_id"`
	PostID    string `json:"post_id"`
}

// Kindはジョブの種類を返す
func (AddPostToTimelineArgs) Kind() string { return "add_post_to_timeline" }

// InsertOptsはジョブのInsertオプションを返す。Priorityは既定 (1, 最高) の
// ままにし、確認メール (Priority 2) など優先度の低いジョブより先にタイムライン配信を
// 処理させる。
func (AddPostToTimelineArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}
}

// exportJobUniqueStatesはエクスポート系ジョブの一意性を判定する状態集合を返す
// (Riverの既定集合からcompletedを除いたもの)。ここでの一意性は「この作業依頼の
// ジョブがまだ待機中または実行中」を意味する必要があり、「最近実行された」ではない。
// このヘルパーを使うジョブはDBの永続状態 (queued行、未通知の成功、最新成功より
// 古いエクスポート) からリコンシリエーションが再導出して再投入するため、行を収束
// させないまま完了したジョブが、job cleanerに消されるまで次の投入を塞いではならない。
//
// Riverが既定集合から安全に外せる状態として挙げているのはretryableだけである。
// completedを外しても投入経路は正しく動く。部分ユニークインデックスは、現在の状態が
// 集合に含まれる行だけを対象とするため。代償として、完了したエクスポート系ジョブを
// 手動で再試行すると、一意性を判定しないまま行がavailableへ戻るため、同じ作業依頼の
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

// GenerateExportMaxAttemptsは、1件のエクスポートを諦めるまでにジョブキューが
// 生成ジョブを実行する回数。リコンシリエーションが同じ回数を必要とするためexported
// にしている。プロセスの異常終了でstartedのまま残ったエクスポートは、試行を使い
// 切って初めてfailedとして閉じられるため、この値とずれた回数を持つと、早すぎる
// 打ち切りか、見込みのないエクスポートの無限の再試行のどちらかになる。
const GenerateExportMaxAttempts = 5

// GenerateExportArgsは1件のエクスポートのzipを生成し、オブジェクト
// ストレージへストリーミングするジョブの引数。
type GenerateExportArgs struct {
	ExportID string `json:"export_id"`
}

// Kindはジョブの種類を返す。
func (GenerateExportArgs) Kind() string { return "generate_export" }

// InsertOptsはジョブのInsertオプションを返す。生成は専用のexportキューで
// 実行し、DBやオブジェクトストレージの一時的な失敗でエクスポートを終わらせないよう
// 最大5回まで再試行する。一意性はエクスポートID単位とし、Createのコミット後の
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

// CleanupOldExportsArgsは1つのプロフィールについて、最新の成功より古い
// エクスポートをオブジェクト → 行の順に削除するジョブの引数。
type CleanupOldExportsArgs struct {
	ProfileID string `json:"profile_id"`
}

// Kindはジョブの種類を返す。
func (CleanupOldExportsArgs) Kind() string { return "cleanup_old_exports" }

// InsertOptsはジョブのInsertオプションを返す。削除はPriority 3 (確認メールの
// 2、タイムライン配信の既定1より低い) に降格する。新しいエクスポートは既に
// ダウンロード可能で、古いものは既にダウンロード対象から外れており、誰も待っていない
// ため。一意性はプロフィール単位とし、成功後の投入とリコンシリエーションの投入を
// 1回の実行にまとめる。
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

// SendExportCompletedEmailArgsはエクスポートがダウンロード可能になったことを
// 申請者へ通知するジョブの引数。
type SendExportCompletedEmailArgs struct {
	ExportID string `json:"export_id"`
}

// Kindはジョブの種類を返す。
func (SendExportCompletedEmailArgs) Kind() string { return "send_export_completed_email" }

// InsertOptsはジョブのInsertオプションを返す。Priorityは確認メールと同じ
// 2とする。どちらもユーザーが待っているtransactionalなメールであるため。
// 一意性はエクスポート単位とし、成功後の投入と未通知エクスポートに対する
// リコンシリエーションの投入を1回の送信にまとめる。
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

// ReconcileExportsArgsは、投入の失敗や処理途中でのプロセス終了によって
// 取り残されたエクスポートを収束させる定期ジョブの引数。フィールドを持たない。
// 未処理の作業はジョブ引数からではなく、実行時にexportsテーブルから導出する。
type ReconcileExportsArgs struct{}

// Kindはジョブの種類を返す。
func (ReconcileExportsArgs) Kind() string { return "reconcile_exports" }

// InsertOptsはジョブのInsertオプションを返す。次回の定期実行が再試行に
// あたるためMaxAttemptsは1とする。その場で再試行すると失敗したジョブが
// retryableのまま残り、一意性判定によって次の定期投入を塞いでしまう。Priorityは
// cleanupと同じ理由で3に降格する (回復処理は保守作業であり、待っているリクエストは
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

// CleanupOrphanExportObjectsPeriodは孤児回収の定期実行間隔と一意性の時間枠を
// 兼ねる。値を共有することで、リーダーの起動時投入と通常スケジュールを同じ日次契約に
// 揃える。ある時間枠で掃除が完了した後は、別のリーダーが投入を試みてもエクスポートの
// プレフィックスを再度一覧しない。
const CleanupOrphanExportObjectsPeriod = 24 * time.Hour

// CleanupOrphanExportObjectsArgsは、どのエクスポートからも保持されなくなった
// エクスポートオブジェクトを削除する定期ジョブの引数。持つのは走査を再開する位置だけ
// である。一覧されたどのオブジェクトが孤児かは、ジョブ引数からではなく、実行時に
// オブジェクトストレージとexportsテーブルを照合することで求める。
type CleanupOrphanExportObjectsArgs struct {
	// StartAfterは走査を再開する位置となるオブジェクトキー。日次スケジュールは
	// これを空で投入し、走査はプレフィックスの先頭から始まる。走査予算を使い切った実行
	// は止まったキーを持つ継続ジョブを投入する。1回の実行の予算に収まらないプレフィックス
	// を、連なった複数回の実行で網羅できるのはこのためである。
	StartAfter string `json:"start_after,omitempty"`
}

// Kindはジョブの種類を返す。
func (CleanupOrphanExportObjectsArgs) Kind() string { return "cleanup_orphan_export_objects" }

// InsertOptsはジョブのInsertオプションを返す。reconcile_exportsとは異なり、
// この掃除はその場で再試行する。スケジュールが日次のため、オブジェクトストレージの
// 一時的な失敗を次回の実行に委ねると孤児オブジェクトの課金がもう1日続くこと、および
// Riverのバックオフでは試行が数分で尽きることによる。一意性の時間枠は日次の実行間隔
// と揃え、状態集合にはcompletedも含める。これにより、選出された各リーダーが起動時に
// 投入しても、同じ時間枠で成功済みの一覧は繰り返さない。Priorityは他の保守ジョブと
// 同じ理由で3に降格する。
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

// JobInserterはジョブをキューに追加するインターフェース
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Dispatcherはジョブキューへの投入を抽象化する
type Dispatcher struct {
	client JobInserter
}

// NewDispatcherは新しいDispatcherを生成する
func NewDispatcher(client JobInserter) *Dispatcher {
	return &Dispatcher{client: client}
}

// DeferredInserterは実体のinserterを構築後に注入するJobInserter。
// DispatcherとRiverクライアントの初期化循環を断つ: ジョブをenqueueする
// UseCase (fanout等) はDispatcherを必要とし、DispatcherはJobInserterを必要と
// するが、JobInserterを満たすRiverクライアントは、そのUseCaseを内包するWorkerを
// 登録した後でないと生成できない。先にDeferredInserterを包んだDispatcherを作り、
// Riverクライアント生成後にSetInserterを呼ぶ。
type DeferredInserter struct {
	inserter JobInserter
}

// SetInserterは実体のinserterを注入する。Riverクライアント生成後・
// ジョブ実行前に一度だけ呼ぶ。
func (d *DeferredInserter) SetInserter(inserter JobInserter) {
	d.inserter = inserter
}

// Insertは注入済みのinserterに委譲する。
func (d *DeferredInserter) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return d.inserter.Insert(ctx, args, opts)
}

// EnqueueEmailConfirmationはメール確認コード送信ジョブをキューに追加する
func (d *Dispatcher) EnqueueEmailConfirmation(ctx context.Context, email, code, locale string) error {
	args := SendEmailConfirmationArgs{Email: email, Code: code, Locale: locale}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueueFanoutPostは投稿を投稿者のフォロワーへ配信するジョブをenqueueする。
// dispatcherがmodelパッケージに依存しないようpost IDは文字列で受け取る
// (呼び出し側がpostID.String() を渡す)。
func (d *Dispatcher) EnqueueFanoutPost(ctx context.Context, postID string) error {
	args := FanoutPostArgs{PostID: postID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueueAddPostToTimelineは投稿を1つのプロフィールのホームタイムラインに
// 追加するジョブをenqueueする。dispatcherをmodel非依存に保つためIDは文字列で
// 受け取る (呼び出し側がprofileID.String() / postID.String() を渡す)。
func (d *Dispatcher) EnqueueAddPostToTimeline(ctx context.Context, profileID, postID string) error {
	args := AddPostToTimelineArgs{ProfileID: profileID, PostID: postID}
	opts := args.InsertOpts()
	_, err := d.client.Insert(ctx, args, &opts)
	return err
}

// EnqueueGenerateExportはエクスポートのzipを生成するジョブをenqueueする。
// dispatcherをmodel非依存に保つためIDは文字列で受け取る (呼び出し側が
// exportID.String() を渡す)。
//
// 実際に投入したかどうかを返す。同じエクスポートのジョブが未完了のまま残っている
// ときの投入に対して、Riverはエラーではなくskip済みの結果を返すため、
// リコンシリエーションはskipされた候補を1回の実行で引き受けた件数に数えては
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

// EnqueueCleanupOldExportsは、最新の成功より古いプロフィールのエクスポートを
// 削除するジョブをenqueueする。dispatcherをmodel非依存に保つためIDは文字列で
// 受け取る (呼び出し側がprofileID.String() を渡す)。
//
// EnqueueGenerateExportと同じ条件で、実際に投入したかどうかを返す。
func (d *Dispatcher) EnqueueCleanupOldExports(ctx context.Context, profileID string) (bool, error) {
	args := CleanupOldExportsArgs{ProfileID: profileID}
	opts := args.InsertOpts()
	res, err := d.client.Insert(ctx, args, &opts)
	if err != nil {
		return false, err
	}
	return !res.UniqueSkippedAsDuplicate, nil
}

// EnqueueSendExportCompletedEmailはエクスポートの完成を申請者へ通知する
// ジョブをenqueueする。dispatcherをmodel非依存に保つためIDは文字列で受け取る
// (呼び出し側がexportID.String() を渡す)。
//
// EnqueueGenerateExportと同じ条件で、実際に投入したかどうかを返す。
func (d *Dispatcher) EnqueueSendExportCompletedEmail(ctx context.Context, exportID string) (bool, error) {
	args := SendExportCompletedEmailArgs{ExportID: exportID}
	opts := args.InsertOpts()
	res, err := d.client.Insert(ctx, args, &opts)
	if err != nil {
		return false, err
	}
	return !res.UniqueSkippedAsDuplicate, nil
}

// EnqueueReconcileExportsは、投入の失敗や処理途中でのプロセス終了によって
// 取り残されたエクスポートを収束させるジョブをenqueueする。
//
// EnqueueGenerateExportと同じ条件で、実際に投入したかどうかを返す。
func (d *Dispatcher) EnqueueReconcileExports(ctx context.Context) (bool, error) {
	args := ReconcileExportsArgs{}
	opts := args.InsertOpts()
	res, err := d.client.Insert(ctx, args, &opts)
	if err != nil {
		return false, err
	}
	return !res.UniqueSkippedAsDuplicate, nil
}

// EnqueueCleanupOrphanExportObjectsは、どのエクスポートからも保持されなくなった
// エクスポートオブジェクトを削除する掃除を、startAfterの次から再開する形でenqueue
// する。startAfterが空ならプレフィックスを先頭から走査する。掃除は走査の残りを
// 引き渡すために、自身が止まったキーを渡す。
//
// EnqueueGenerateExportと同じ条件で、実際に投入したかどうかを返す。
func (d *Dispatcher) EnqueueCleanupOrphanExportObjects(ctx context.Context, startAfter string) (bool, error) {
	args := CleanupOrphanExportObjectsArgs{StartAfter: startAfter}
	opts := args.InsertOpts()
	res, err := d.client.Insert(ctx, args, &opts)
	if err != nil {
		return false, err
	}
	return !res.UniqueSkippedAsDuplicate, nil
}
