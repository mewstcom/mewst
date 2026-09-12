// Package worker はバックグラウンドワーカー機能を提供します
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/email"
	mewstsentry "github.com/mewstcom/mewst/go/internal/sentry"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// queueConfigs returns the queues this worker serves. The export queue runs a
// single worker: generation streams a whole archive from the database to the
// object storage, so running several at once would multiply the memory and
// bandwidth one process needs, and one profile's long export would still finish
// while the others wait rather than all of them slowing down together.
//
// [Ja] queueConfigs は本 worker が処理するキューを返す。export キューは worker を
// 1 つだけ動かす。生成は DB からオブジェクトストレージへアーカイブ全体を
// ストリーミングするため、同時に何本も走らせると 1 プロセスに必要なメモリと帯域が
// その数だけ増える。直列化しても、あるプロフィールの長いエクスポートが終わるまで
// 他が待つだけで、全体が一斉に遅くなることはない。
func queueConfigs() map[string]river.QueueConfig {
	return map[string]river.QueueConfig{
		river.QueueDefault:     {MaxWorkers: 10},
		dispatcher.QueueExport: {MaxWorkers: 1},
	}
}

// reconcileExportsInterval is how often the export reconciliation runs. It is
// the upper bound on how long an export whose immediate job insert was lost
// waits before the reconciliation picks it up, so it is kept well below the
// point where a user would take the export for failed.
//
// usecase.ReconcileExportsTimeout stays strictly below this, so a run that
// stopped making progress releases its worker before the next insert instead of
// making that insert skip on the running state's uniqueness.
//
// [Ja] reconcileExportsInterval はエクスポートのリコンシリエーションを実行する間隔。
// 即時のジョブ投入が失われたエクスポートがリコンシリエーションに拾われるまでの待ち時間
// の上限にあたるため、ユーザーがエクスポートを失敗したと受け取る時間より十分短くする。
//
// usecase.ReconcileExportsTimeout はこの値より厳密に小さく保つ。前進しなくなった実行が
// 次の投入より前に worker を解放し、running の一意性でその投入を skip させないため。
const reconcileExportsInterval = 5 * time.Minute

// ExportUsecases groups the export UseCases this client runs. The zero value
// means the object storage is not configured: every export Worker is then left
// unregistered and neither export periodic job is scheduled, so the process
// still starts and serves every other job. They are built together, so they are
// absent together.
//
// [Ja] ExportUsecases は本クライアントが実行するエクスポート系 UseCase をまとめる。
// ゼロ値はオブジェクトストレージが未設定であることを意味し、その場合はエクスポート系
// Worker を登録せず、エクスポートの定期ジョブも登録しない。これにより、プロセスは起動
// して他のジョブを引き続き処理する。これらはまとめて構築されるため、欠ける場合もまとめて
// 欠ける。
type ExportUsecases struct {
	Generate             *usecase.GenerateExportUsecase
	CleanupOld           *usecase.CleanupOldExportsUsecase
	SendCompletedEmail   *usecase.SendExportCompletedEmailUsecase
	Reconcile            *usecase.ReconcileExportsUsecase
	CleanupOrphanObjects *usecase.CleanupOrphanExportObjectsUsecase
}

// configured reports whether the export UseCases were built, which they are
// only when the object storage is configured.
//
// [Ja] configured はエクスポート系 UseCase が構築済みかどうかを返す。これらが構築
// されるのはオブジェクトストレージが設定されている場合だけである。
func (u ExportUsecases) configured() bool {
	return u.Generate != nil &&
		u.CleanupOld != nil &&
		u.SendCompletedEmail != nil &&
		u.Reconcile != nil &&
		u.CleanupOrphanObjects != nil
}

// periodicJobSpec is one scheduled insert: the arguments to insert, how often,
// and whether a newly elected leader inserts one immediately. River's
// PeriodicJob keeps all three behind unexported fields, so the schedule is
// declared here first and converted afterwards, which also lets a test read
// what was scheduled.
//
// [Ja] periodicJobSpec は定期投入 1 件分の設定。投入する引数、間隔、そして新しく
// 選出されたリーダーが直ちに 1 件投入するかどうかを持つ。River の PeriodicJob は
// これら 3 つを非公開フィールドに持つため、スケジュールを先にここで宣言してから変換
// する。これにより、何をスケジュールしたかをテストから読めるようにもなる。
type periodicJobSpec struct {
	args       river.JobArgs
	interval   time.Duration
	runOnStart bool
}

// exportPeriodicJobSpecs returns the schedules that drive export recovery, or
// nothing when the export UseCases are absent. Scheduling a periodic job whose
// Worker is not registered would insert jobs nothing can work, so the schedule
// follows the same gate as the registration.
//
// [Ja] exportPeriodicJobSpecs はエクスポートの回復処理を駆動するスケジュールを返す。
// エクスポート系 UseCase が無い場合は何も返さない。Worker を登録していない定期ジョブを
// 登録すると、誰も処理できないジョブを投入し続けることになるため、スケジュールも登録と
// 同じゲートに従わせる。
func exportPeriodicJobSpecs(exportUCs ExportUsecases) []periodicJobSpec {
	if !exportUCs.configured() {
		return nil
	}

	return []periodicJobSpec{
		// Running on start makes a deployment or a crash a trigger for
		// reconciliation as well: the schedule only lives in the elected leader's
		// memory, so a process that just took over would otherwise wait a full
		// interval before recovering what the previous one left behind.
		//
		// [Ja] 起動時実行により、デプロイやクラッシュもリコンシリエーションの
		// きっかけになる。スケジュールは選出されたリーダーのメモリ上にしか存在しない
		// ため、引き継いだ直後のプロセスは、これが無いと前のプロセスが残した処理を
		// 回復するまで 1 間隔分待つことになる。
		{
			args:       dispatcher.ReconcileExportsArgs{},
			interval:   reconcileExportsInterval,
			runOnStart: true,
		},
		// Run on every elected leader's start because the periodic schedule only
		// lives in leader memory and could otherwise be postponed forever by
		// repeated leadership changes. The Args' daily uniqueness window includes
		// completed jobs, so another leader in the same window skips the insert
		// instead of repeating the full-prefix listing.
		//
		// [Ja] 定期スケジュールはリーダーのメモリにしか存在せず、リーダー交代が続くと
		// 実行が無期限に先送りされうるため、選出された各リーダーの起動時に投入する。
		// Args の日次の一意性には completed も含まれるため、同じ時間枠の別リーダーは
		// 投入を skip し、プレフィックスの全件一覧を繰り返さない。
		{
			args:       dispatcher.CleanupOrphanExportObjectsArgs{},
			interval:   dispatcher.CleanupOrphanExportObjectsPeriod,
			runOnStart: true,
		},
	}
}

// periodicJobConstructor returns what River calls to build each insert of a
// scheduled job. It returns no insert options, so every insert takes the ones
// the Args type declares, including either work-intent uniqueness or the orphan
// sweep's daily completed-run suppression.
//
// It is a named function rather than a literal inside toPeriodicJobs because
// river.PeriodicJob keeps the constructor behind an unexported field: this is
// the only place a test can read back what a schedule inserts.
//
// [Ja] periodicJobConstructor は、スケジュールされたジョブの投入ごとに River が呼ぶ
// ものを返す。insert オプションは返さないため、各投入は Args 型が宣言するオプションを
// 使う。未完了の作業依頼をまとめる一意性、または孤児回収の同じ日次時間枠における
// 成功済み実行の抑止もそこに含まれる。
//
// toPeriodicJobs 内のリテラルではなく名前付き関数にしているのは、river.PeriodicJob が
// コンストラクタを非公開フィールドに持つためである。スケジュールが何を投入するかを
// テストから読み戻せるのはここだけになる。
func periodicJobConstructor(spec periodicJobSpec) func() (river.JobArgs, *river.InsertOpts) {
	return func() (river.JobArgs, *river.InsertOpts) {
		return spec.args, nil
	}
}

// toPeriodicJobs converts the schedules into River's periodic jobs.
//
// [Ja] toPeriodicJobs はスケジュールを River の定期ジョブへ変換する。
func toPeriodicJobs(specs []periodicJobSpec) []*river.PeriodicJob {
	jobs := make([]*river.PeriodicJob, 0, len(specs))
	for _, spec := range specs {
		jobs = append(jobs, river.NewPeriodicJob(
			river.PeriodicInterval(spec.interval),
			periodicJobConstructor(spec),
			&river.PeriodicJobOpts{RunOnStart: spec.runOnStart},
		))
	}
	return jobs
}

// registerWorkers builds the set of Workers this client serves. A UseCase that
// cannot be built for the current configuration is passed as nil and its Worker
// is left out, so the process still starts and serves every job it can.
//
// [Ja] registerWorkers は本クライアントが処理する Worker の集合を構築する。現在の
// 設定では構築できない UseCase は nil で渡され、その Worker は登録されない。これに
// より、プロセスは起動して処理できるジョブを引き続き処理する。
func registerWorkers(
	ctx context.Context,
	sendEmailConfirmationUC *usecase.SendEmailConfirmationUsecase,
	fanoutPostUC *usecase.FanoutPostUsecase,
	addPostToTimelineUC *usecase.AddPostToTimelineUsecase,
	exportUCs ExportUsecases,
) *river.Workers {
	workers := river.NewWorkers()

	// Email delivery.
	//
	// [Ja] メール送信。
	river.AddWorker(workers, NewSendEmailConfirmationWorker(sendEmailConfirmationUC))
	slog.InfoContext(ctx, "SendEmailConfirmationWorker を登録しました")

	// Timeline delivery.
	//
	// [Ja] タイムライン配信。
	river.AddWorker(workers, NewFanoutPostWorker(fanoutPostUC))
	slog.InfoContext(ctx, "FanoutPostWorker を登録しました")
	river.AddWorker(workers, NewAddPostToTimelineWorker(addPostToTimelineUC))
	slog.InfoContext(ctx, "AddPostToTimelineWorker を登録しました")

	// Export generation and its recovery.
	//
	// [Ja] エクスポートの生成とその回復処理。
	if !exportUCs.configured() {
		slog.WarnContext(ctx, "オブジェクトストレージが設定されていないため、エクスポート機能は無効です")
		return workers
	}

	river.AddWorker(workers, NewGenerateExportWorker(exportUCs.Generate))
	slog.InfoContext(ctx, "GenerateExportWorker を登録しました")
	river.AddWorker(workers, NewCleanupOldExportsWorker(exportUCs.CleanupOld))
	slog.InfoContext(ctx, "CleanupOldExportsWorker を登録しました")
	river.AddWorker(workers, NewSendExportCompletedEmailWorker(exportUCs.SendCompletedEmail))
	slog.InfoContext(ctx, "SendExportCompletedEmailWorker を登録しました")
	river.AddWorker(workers, NewReconcileExportsWorker(exportUCs.Reconcile))
	slog.InfoContext(ctx, "ReconcileExportsWorker を登録しました")
	river.AddWorker(workers, NewCleanupOrphanExportObjectsWorker(exportUCs.CleanupOrphanObjects))
	slog.InfoContext(ctx, "CleanupOrphanExportObjectsWorker を登録しました")

	return workers
}

// Client は River クライアントのラッパー
type Client struct {
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

// NewClient creates a new River client. fanoutPostUC / addPostToTimelineUC and
// the export UseCases depend on repository, so they cannot be built inside the
// worker package (worker is forbidden by depguard from importing repository /
// query); build them in main.go and inject them here.
//
// emailSender is injected for the same reason one step removed: the export
// completion mail is sent by a UseCase that reads the notification outbox, so
// that UseCase is built in main.go and needs the sender there. Building the
// sender here as well would put the "deliver or discard" decision in two
// places.
//
// exportUCs is the zero value when the object storage is not configured. The
// export flow has nowhere to upload to then, so its Workers are left
// unregistered, its periodic jobs are not scheduled, and the process still
// starts and serves every other job.
//
// [Ja] NewClient は新しい River クライアントを作成する。fanoutPostUC /
// addPostToTimelineUC とエクスポート系 UseCase は repository に依存するため worker 内
// では構築できず (worker は depguard で repository / query への依存が禁止)、main.go で
// 構築して注入する。
//
// emailSender を注入するのも、一段隔てた同じ理由による。エクスポート完了メールを送るのは
// 通知 outbox を読む UseCase であり、その UseCase は main.go で構築されるため sender も
// そちらで必要になる。ここでも sender を構築すると「配信するか捨てるか」の判断が 2 箇所に
// できてしまう。
//
// exportUCs はオブジェクトストレージが未設定のときゼロ値になる。その場合エクスポートの
// 処理にはアップロード先が無いため、その Worker は登録せず、定期ジョブも登録しない。
// プロセスは起動して他のジョブを処理し続ける。
func NewClient(
	ctx context.Context,
	databaseURL string,
	emailSender email.Sender,
	fanoutPostUC *usecase.FanoutPostUsecase,
	addPostToTimelineUC *usecase.AddPostToTimelineUsecase,
	exportUCs ExportUsecases,
) (*Client, error) {
	// pgxpool の作成
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	// コネクションプール設定
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 5 * time.Minute
	poolConfig.MaxConnIdleTime = 2 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	confirmationSender := email.NewConfirmationSender(emailSender)
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(confirmationSender)
	workers := registerWorkers(ctx, sendEmailConfirmationUC, fanoutPostUC, addPostToTimelineUC, exportUCs)

	// River クライアントの作成
	// Middleware には Sentry エラーキャプチャ用の WorkerMiddleware を登録する。
	// これにより全 Worker のジョブ失敗が自動的に Sentry に送信される。
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:       queueConfigs(),
		Workers:      workers,
		PeriodicJobs: toPeriodicJobs(exportPeriodicJobSpecs(exportUCs)),
		Logger:       slog.Default(),
		Middleware: []rivertype.Middleware{
			mewstsentry.RiverWorkerMiddleware(),
		},
	})
	if err != nil {
		pool.Close()
		return nil, err
	}

	return &Client{
		riverClient: riverClient,
		pool:        pool,
	}, nil
}

// Start は River クライアントを起動します
func (c *Client) Start(ctx context.Context) error {
	slog.InfoContext(ctx, "River クライアントを起動します")
	return c.riverClient.Start(ctx)
}

// Stop は River クライアントを停止します
func (c *Client) Stop(ctx context.Context) error {
	slog.InfoContext(ctx, "River クライアントを停止します")
	if err := c.riverClient.Stop(ctx); err != nil {
		return err
	}
	c.pool.Close()
	return nil
}

// Client は River クライアントへのアクセスを提供します
func (c *Client) Client() *river.Client[pgx.Tx] {
	return c.riverClient
}
