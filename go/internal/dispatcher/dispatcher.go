// Package dispatcher はジョブキューへの投入を抽象化する。
// Repository がデータベースアクセスを抽象化するのと同じ発想で、
// Dispatcher がジョブキューアクセスを抽象化する。
package dispatcher

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

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
