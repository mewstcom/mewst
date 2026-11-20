// Package workerはバックグラウンドワーカー機能を提供します
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

// queueConfigsは本workerが処理するキューを返す。exportキューはworkerを
// 1つだけ動かす。生成はDBからオブジェクトストレージへアーカイブ全体を
// ストリーミングするため、同時に何本も走らせると1プロセスに必要なメモリと帯域が
// その数だけ増える。直列化しても、あるプロフィールの長いエクスポートが終わるまで
// 他が待つだけで、全体が一斉に遅くなることはない。
func queueConfigs() map[string]river.QueueConfig {
	return map[string]river.QueueConfig{
		river.QueueDefault:     {MaxWorkers: 10},
		dispatcher.QueueExport: {MaxWorkers: 1},
	}
}

// reconcileExportsIntervalはエクスポートのリコンシリエーションを実行する間隔。
// 即時のジョブ投入が失われたエクスポートがリコンシリエーションに拾われるまでの待ち時間
// の上限にあたるため、ユーザーがエクスポートを失敗したと受け取る時間より十分短くする。
//
// usecase.ReconcileExportsTimeoutはこの値より厳密に小さく保つ。前進しなくなった実行が
// 次の投入より前にworkerを解放し、runningの一意性でその投入をskipさせないため。
const reconcileExportsInterval = 5 * time.Minute

// ExportUsecasesは本クライアントが実行するエクスポート系UseCaseをまとめる。
// ゼロ値はオブジェクトストレージが未設定であることを意味し、その場合はエクスポート系
// Workerを登録せず、エクスポートの定期ジョブも登録しない。これにより、プロセスは起動
// して他のジョブを引き続き処理する。これらはまとめて構築されるため、欠ける場合もまとめて
// 欠ける。
type ExportUsecases struct {
	Generate             *usecase.GenerateExportUsecase
	CleanupOld           *usecase.CleanupOldExportsUsecase
	SendCompletedEmail   *usecase.SendExportCompletedEmailUsecase
	Reconcile            *usecase.ReconcileExportsUsecase
	CleanupOrphanObjects *usecase.CleanupOrphanExportObjectsUsecase
}

// configuredはエクスポート系UseCaseが構築済みかどうかを返す。これらが構築
// されるのはオブジェクトストレージが設定されている場合だけである。
func (u ExportUsecases) configured() bool {
	return u.Generate != nil &&
		u.CleanupOld != nil &&
		u.SendCompletedEmail != nil &&
		u.Reconcile != nil &&
		u.CleanupOrphanObjects != nil
}

// periodicJobSpecは定期投入1件分の設定。投入する引数、間隔、そして新しく
// 選出されたリーダーが直ちに1件投入するかどうかを持つ。RiverのPeriodicJobは
// これら3つを非公開フィールドに持つため、スケジュールを先にここで宣言してから変換
// する。これにより、何をスケジュールしたかをテストから読めるようにもなる。
type periodicJobSpec struct {
	args       river.JobArgs
	interval   time.Duration
	runOnStart bool
}

// exportPeriodicJobSpecsはエクスポートの回復処理を駆動するスケジュールを返す。
// エクスポート系UseCaseが無い場合は何も返さない。Workerを登録していない定期ジョブを
// 登録すると、誰も処理できないジョブを投入し続けることになるため、スケジュールも登録と
// 同じゲートに従わせる。
func exportPeriodicJobSpecs(exportUCs ExportUsecases) []periodicJobSpec {
	if !exportUCs.configured() {
		return nil
	}

	return []periodicJobSpec{
		// 起動時実行により、デプロイやクラッシュもリコンシリエーションの
		// きっかけになる。スケジュールは選出されたリーダーのメモリ上にしか存在しない
		// ため、引き継いだ直後のプロセスは、これが無いと前のプロセスが残した処理を
		// 回復するまで1間隔分待つことになる。
		{
			args:       dispatcher.ReconcileExportsArgs{},
			interval:   reconcileExportsInterval,
			runOnStart: true,
		},
		// 定期スケジュールはリーダーのメモリにしか存在せず、リーダー交代が続くと
		// 実行が無期限に先送りされうるため、選出された各リーダーの起動時に投入する。
		// Argsの日次の一意性にはcompletedも含まれるため、同じ時間枠の別リーダーは
		// 投入をskipし、プレフィックスの全件一覧を繰り返さない。
		{
			args:       dispatcher.CleanupOrphanExportObjectsArgs{},
			interval:   dispatcher.CleanupOrphanExportObjectsPeriod,
			runOnStart: true,
		},
	}
}

// periodicJobConstructorは、スケジュールされたジョブの投入ごとにRiverが呼ぶ
// ものを返す。insertオプションは返さないため、各投入はArgs型が宣言するオプションを
// 使う。未完了の作業依頼をまとめる一意性、または孤児回収の同じ日次時間枠における
// 成功済み実行の抑止もそこに含まれる。
//
// toPeriodicJobs内のリテラルではなく名前付き関数にしているのは、river.PeriodicJobが
// コンストラクタを非公開フィールドに持つためである。スケジュールが何を投入するかを
// テストから読み戻せるのはここだけになる。
func periodicJobConstructor(spec periodicJobSpec) func() (river.JobArgs, *river.InsertOpts) {
	return func() (river.JobArgs, *river.InsertOpts) {
		return spec.args, nil
	}
}

// toPeriodicJobsはスケジュールをRiverの定期ジョブへ変換する。
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

// registerWorkersは本クライアントが処理するWorkerの集合を構築する。現在の
// 設定では構築できないUseCaseはnilで渡され、そのWorkerは登録されない。これに
// より、プロセスは起動して処理できるジョブを引き続き処理する。
func registerWorkers(
	ctx context.Context,
	sendEmailConfirmationUC *usecase.SendEmailConfirmationUsecase,
	fanoutPostUC *usecase.FanoutPostUsecase,
	addPostToTimelineUC *usecase.AddPostToTimelineUsecase,
	exportUCs ExportUsecases,
) *river.Workers {
	workers := river.NewWorkers()

	// メール送信。
	river.AddWorker(workers, NewSendEmailConfirmationWorker(sendEmailConfirmationUC))
	slog.InfoContext(ctx, "SendEmailConfirmationWorkerを登録しました")

	// タイムライン配信。
	river.AddWorker(workers, NewFanoutPostWorker(fanoutPostUC))
	slog.InfoContext(ctx, "FanoutPostWorkerを登録しました")
	river.AddWorker(workers, NewAddPostToTimelineWorker(addPostToTimelineUC))
	slog.InfoContext(ctx, "AddPostToTimelineWorkerを登録しました")

	// エクスポートの生成とその回復処理。
	if !exportUCs.configured() {
		slog.WarnContext(ctx, "オブジェクトストレージが設定されていないため、エクスポート機能は無効です")
		return workers
	}

	river.AddWorker(workers, NewGenerateExportWorker(exportUCs.Generate))
	slog.InfoContext(ctx, "GenerateExportWorkerを登録しました")
	river.AddWorker(workers, NewCleanupOldExportsWorker(exportUCs.CleanupOld))
	slog.InfoContext(ctx, "CleanupOldExportsWorkerを登録しました")
	river.AddWorker(workers, NewSendExportCompletedEmailWorker(exportUCs.SendCompletedEmail))
	slog.InfoContext(ctx, "SendExportCompletedEmailWorkerを登録しました")
	river.AddWorker(workers, NewReconcileExportsWorker(exportUCs.Reconcile))
	slog.InfoContext(ctx, "ReconcileExportsWorkerを登録しました")
	river.AddWorker(workers, NewCleanupOrphanExportObjectsWorker(exportUCs.CleanupOrphanObjects))
	slog.InfoContext(ctx, "CleanupOrphanExportObjectsWorkerを登録しました")

	return workers
}

// ClientはRiverクライアントのラッパー
type Client struct {
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

// NewClientは新しいRiverクライアントを作成する。fanoutPostUC /
// addPostToTimelineUCとエクスポート系UseCaseはrepositoryに依存するためworker内
// では構築できず (workerはdepguardでrepository / queryへの依存が禁止)、main.goで
// 構築して注入する。
//
// emailSenderを注入するのも、一段隔てた同じ理由による。エクスポート完了メールを送るのは
// 通知outboxを読むUseCaseであり、そのUseCaseはmain.goで構築されるためsenderも
// そちらで必要になる。ここでもsenderを構築すると「配信するか捨てるか」の判断が2箇所に
// できてしまう。
//
// exportUCsはオブジェクトストレージが未設定のときゼロ値になる。その場合エクスポートの
// 処理にはアップロード先が無いため、そのWorkerは登録せず、定期ジョブも登録しない。
// プロセスは起動して他のジョブを処理し続ける。
func NewClient(
	ctx context.Context,
	databaseURL string,
	emailSender email.Sender,
	fanoutPostUC *usecase.FanoutPostUsecase,
	addPostToTimelineUC *usecase.AddPostToTimelineUsecase,
	exportUCs ExportUsecases,
) (*Client, error) {
	// pgxpoolの作成
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

	// Riverクライアントの作成
	// MiddlewareにはSentryエラーキャプチャ用のWorkerMiddlewareを登録する。
	// これにより全Workerのジョブ失敗が自動的にSentryに送信される。
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

// StartはRiverクライアントを起動します
func (c *Client) Start(ctx context.Context) error {
	slog.InfoContext(ctx, "Riverクライアントを起動します")
	return c.riverClient.Start(ctx)
}

// StopはRiverクライアントを停止します
func (c *Client) Stop(ctx context.Context) error {
	slog.InfoContext(ctx, "Riverクライアントを停止します")
	if err := c.riverClient.Stop(ctx); err != nil {
		return err
	}
	c.pool.Close()
	return nil
}

// ClientはRiverクライアントへのアクセスを提供します
func (c *Client) Client() *river.Client[pgx.Tx] {
	return c.riverClient
}
