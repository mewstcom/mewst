package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

type recordingGenerateExportExecutor struct {
	inputs []usecase.GenerateExportInput
}

func (executor *recordingGenerateExportExecutor) Execute(_ context.Context, input usecase.GenerateExportInput) error {
	executor.inputs = append(executor.inputs, input)
	return nil
}

type recordingCleanupOldExportsExecutor struct {
	profileIDs []model.ProfileID
	err        error
}

func (executor *recordingCleanupOldExportsExecutor) Execute(_ context.Context, profileID model.ProfileID) error {
	executor.profileIDs = append(executor.profileIDs, profileID)
	return executor.err
}

type recordingSendExportCompletedEmailExecutor struct {
	exportIDs []model.ExportID
	err       error
}

func (executor *recordingSendExportCompletedEmailExecutor) Execute(_ context.Context, exportID model.ExportID) error {
	executor.exportIDs = append(executor.exportIDs, exportID)
	return executor.err
}

func TestQueueConfigs(t *testing.T) {
	t.Parallel()

	configs := queueConfigs()

	// MaxWorkers is per River client, so this limits generation to one at a time
	// per process: it streams a whole archive from the database to the object
	// storage, and concurrent runs would multiply the memory and bandwidth one
	// process needs.
	//
	// [Ja] MaxWorkers は River クライアント単位の設定であり、ここでは生成を
	// 1 プロセスにつき 1 本に制限している。生成は DB からオブジェクトストレージへ
	// アーカイブ全体をストリーミングするため、同時実行すると 1 プロセスに必要な
	// メモリと帯域がその数だけ増える。
	exportQueue, ok := configs[dispatcher.QueueExport]
	if !ok {
		t.Fatalf("%q キューが登録されていません", dispatcher.QueueExport)
	}
	if exportQueue.MaxWorkers != 1 {
		t.Errorf("%q キューの MaxWorkers = %d, want 1", dispatcher.QueueExport, exportQueue.MaxWorkers)
	}

	// The export queue must be an addition, not a replacement: the default queue
	// still runs timeline delivery and the emails.
	//
	// [Ja] export キューは追加であって置き換えではない。既定キューは引き続き
	// タイムライン配信とメール送信を処理する。
	defaultQueue, ok := configs[river.QueueDefault]
	if !ok {
		t.Fatalf("%q キューが登録されていません", river.QueueDefault)
	}
	if defaultQueue.MaxWorkers != 10 {
		t.Errorf("%q キューの MaxWorkers = %d, want 10", river.QueueDefault, defaultQueue.MaxWorkers)
	}
}

// jobArgsWithInsertOpts is what an Args type must satisfy to be covered by
// TestJobQueuesAreServed: Kind names the subtest and InsertOpts names the
// queue. Declaring the table with this type turns an Args type that forgets
// InsertOpts into a compile error rather than a failure at test time.
//
// [Ja] jobArgsWithInsertOpts は TestJobQueuesAreServed が対象にする Args 型が
// 満たすべき interface。Kind はサブテスト名に、InsertOpts はキュー名に使う。この型で
// テーブルを宣言することで、InsertOpts を持たない Args 型はテスト実行時ではなく
// コンパイル時に検出される。
type jobArgsWithInsertOpts interface {
	river.JobArgs
	river.JobArgsWithInsertOpts
}

// TestJobQueuesAreServed guards against a job being inserted into a queue this
// worker never polls. River accepts such an insert, so the job would sit
// available forever with nothing to signal it. Every Args type the dispatcher
// defines is listed, not only the export ones: now that the worker serves more
// than one queue, moving any job to a queue that queueConfigs forgets has the
// same outcome.
//
// [Ja] TestJobQueuesAreServed は、本 worker が polling しないキューへジョブが投入される
// 事態を防ぐ。River はその投入を受け付けてしまうため、ジョブは何の兆候もなく available
// のまま残り続ける。対象は dispatcher が定義する全 Args 型とし、エクスポート系に限らない。
// worker が複数のキューを処理するようになった以上、どのジョブでも queueConfigs が
// 取りこぼしたキューへ移せば同じ結果になるため。
func TestJobQueuesAreServed(t *testing.T) {
	t.Parallel()

	configs := queueConfigs()

	jobArgs := []jobArgsWithInsertOpts{
		dispatcher.SendEmailConfirmationArgs{},
		dispatcher.FanoutPostArgs{},
		dispatcher.AddPostToTimelineArgs{},
		dispatcher.GenerateExportArgs{},
		dispatcher.CleanupOldExportsArgs{},
		dispatcher.SendExportCompletedEmailArgs{},
		dispatcher.ReconcileExportsArgs{},
		dispatcher.CleanupOrphanExportObjectsArgs{},
	}

	for _, args := range jobArgs {
		t.Run(args.Kind(), func(t *testing.T) {
			t.Parallel()

			queue := args.InsertOpts().Queue
			if _, served := configs[queue]; !served {
				t.Errorf("%q キューが Worker クライアントに登録されていません", queue)
			}
		})
	}
}

// configuredExportUsecases returns an export bundle that stands for a
// deployment with the object storage configured. The UseCases are empty values
// because these tests only observe whether a Worker was registered, never run
// one.
//
// [Ja] configuredExportUsecases は、オブジェクトストレージが設定されたデプロイを
// 表すエクスポート系の束を返す。これらのテストは Worker が登録されたかどうかを観測する
// だけで実行はしないため、UseCase は空の値でよい。
func configuredExportUsecases() ExportUsecases {
	return ExportUsecases{
		Generate:             &usecase.GenerateExportUsecase{},
		CleanupOld:           &usecase.CleanupOldExportsUsecase{},
		SendCompletedEmail:   &usecase.SendExportCompletedEmailUsecase{},
		Reconcile:            &usecase.ReconcileExportsUsecase{},
		CleanupOrphanObjects: &usecase.CleanupOrphanExportObjectsUsecase{},
	}
}

// TestRegisterWorkers_Export pins the object storage gate: with no storage
// configured there is nowhere to upload an archive to and no export row can be
// created at all, so every export Worker must be left out and the process must
// still serve every other job.
//
// Registration is observed by registering the same Worker again: River rejects
// a second Worker for one job kind, so a nil error means the kind was still
// free and an error means it was already taken.
//
// [Ja] TestRegisterWorkers_Export はオブジェクトストレージによるゲートを固定する。
// ストレージが未設定ならアーカイブのアップロード先が無く、そもそもエクスポート行を
// 作成できないため、エクスポート系 Worker はすべて登録せず、プロセスは他のジョブを
// 引き続き処理できる必要がある。
//
// 登録の有無は同じ Worker を再登録して観測する。River は 1 つのジョブ種別に 2 つ目の
// Worker を拒否するため、エラーが nil ならその種別は空いており、エラーが返れば既に
// 使われている。
func TestRegisterWorkers_Export(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		exportUCs      ExportUsecases
		wantRegistered bool
	}{
		{
			name:           "オブジェクトストレージ未設定ならエクスポート系 Worker を登録しない",
			exportUCs:      ExportUsecases{},
			wantRegistered: false,
		},
		{
			name:           "オブジェクトストレージ設定済みならエクスポート系 Worker を登録する",
			exportUCs:      configuredExportUsecases(),
			wantRegistered: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			workers := registerWorkers(context.Background(), nil, nil, nil, tt.exportUCs)

			reregister := map[string]func() error{
				"GenerateExportWorker": func() error {
					return river.AddWorkerSafely(workers, NewGenerateExportWorker(tt.exportUCs.Generate))
				},
				"CleanupOldExportsWorker": func() error {
					return river.AddWorkerSafely(workers, NewCleanupOldExportsWorker(tt.exportUCs.CleanupOld))
				},
				"SendExportCompletedEmailWorker": func() error {
					return river.AddWorkerSafely(workers, NewSendExportCompletedEmailWorker(tt.exportUCs.SendCompletedEmail))
				},
				"ReconcileExportsWorker": func() error {
					return river.AddWorkerSafely(workers, NewReconcileExportsWorker(tt.exportUCs.Reconcile))
				},
				"CleanupOrphanExportObjectsWorker": func() error {
					return river.AddWorkerSafely(workers, NewCleanupOrphanExportObjectsWorker(tt.exportUCs.CleanupOrphanObjects))
				},
			}

			for name, add := range reregister {
				err := add()
				if registered := err != nil; registered != tt.wantRegistered {
					t.Errorf("%s registered = %v, want %v (AddWorkerSafely() error = %v)", name, registered, tt.wantRegistered, err)
				}
			}
		})
	}
}

// TestExportPeriodicJobSpecs pins the schedules that drive export recovery.
// Nothing else inserts these two jobs, so a schedule that is missing turns the
// recovery off silently: exports whose job insert was lost would stay queued,
// and orphan objects would be billed forever.
//
// [Ja] TestExportPeriodicJobSpecs はエクスポートの回復処理を駆動するスケジュールを
// 固定する。この 2 つのジョブを投入するものは他に無いため、スケジュールが欠けると回復
// 処理が黙って止まる。ジョブの投入が失われたエクスポートは queued のまま残り、孤児
// オブジェクトは永久に課金され続けることになる。
func TestExportPeriodicJobSpecs(t *testing.T) {
	t.Parallel()

	t.Run("オブジェクトストレージ設定済みなら 2 つのスケジュールを登録する", func(t *testing.T) {
		t.Parallel()

		got := exportPeriodicJobSpecs(configuredExportUsecases())

		want := []periodicJobSpec{
			{args: dispatcher.ReconcileExportsArgs{}, interval: 5 * time.Minute, runOnStart: true},
			{args: dispatcher.CleanupOrphanExportObjectsArgs{}, interval: 24 * time.Hour, runOnStart: true},
		}
		if len(got) != len(want) {
			t.Fatalf("スケジュールの件数 = %d, want %d", len(got), len(want))
		}
		for i, wantSpec := range want {
			if got[i] != wantSpec {
				t.Errorf("スケジュール[%d] = %+v, want %+v", i, got[i], wantSpec)
			}
		}
	})

	// A scheduled job whose Worker is not registered would be inserted on every
	// interval and sit available with nothing to work it, so the schedule follows
	// the same gate as the registration.
	//
	// [Ja] Worker を登録していないジョブをスケジュールすると、間隔ごとに投入され、
	// 処理するものが無いまま available で残り続ける。そのためスケジュールは登録と
	// 同じゲートに従う。
	t.Run("オブジェクトストレージ未設定ならスケジュールを登録しない", func(t *testing.T) {
		t.Parallel()

		if got := exportPeriodicJobSpecs(ExportUsecases{}); len(got) != 0 {
			t.Errorf("スケジュール = %+v, want none", got)
		}
	})

	t.Run("River の定期ジョブへ変換する", func(t *testing.T) {
		t.Parallel()

		specs := exportPeriodicJobSpecs(configuredExportUsecases())
		if got := toPeriodicJobs(specs); len(got) != len(specs) {
			t.Fatalf("定期ジョブの件数 = %d, want %d", len(got), len(specs))
		}

		// river.PeriodicJob keeps the constructor behind an unexported field, so
		// what a schedule inserts is read back through the constructor itself.
		//
		// [Ja] river.PeriodicJob はコンストラクタを非公開フィールドに持つため、
		// スケジュールが何を投入するかはコンストラクタ自体を通して読み戻す。
		wantKinds := []string{"reconcile_exports", "cleanup_orphan_export_objects"}
		if len(specs) != len(wantKinds) {
			t.Fatalf("スケジュールの件数 = %d, want %d", len(specs), len(wantKinds))
		}
		for i, spec := range specs {
			args, opts := periodicJobConstructor(spec)()
			if got := args.Kind(); got != wantKinds[i] {
				t.Errorf("コンストラクタ[%d] の Kind = %q, want %q", i, got, wantKinds[i])
			}
			// Returning no options is what makes the insert take the ones the Args
			// type declares. Any options value here, even an empty one, would drop
			// the uniqueness that keeps a run-on-start insert from repeating work
			// already done in the same window.
			//
			// [Ja] オプションを返さないことにより、投入は Args 型が宣言するオプションを
			// 使う。ここで何らかのオプションを返すと、空の値であっても、同じ時間枠で
			// 済んだ処理を起動時投入が繰り返さないための一意性が失われる。
			if opts != nil {
				t.Errorf("コンストラクタ[%d] の opts = %+v, want nil", i, opts)
			}
		}
	})
}

// TestReconcileExportsTimeoutFitsInterval keeps the reconciliation's timeout
// below the interval it is scheduled at. The job is unique across the running
// state, so a run still holding its worker when the next periodic insert
// arrives makes that insert skip, and the recovery a stalled run already
// delayed would wait another interval. The two values live in different
// packages, so nothing but this test holds them in relation.
//
// [Ja] TestReconcileExportsTimeoutFitsInterval は、リコンシリエーションの timeout を
// 定期実行間隔より小さく保つ。このジョブは running を含む状態で一意なため、次の定期投入
// の時点でまだ worker を保持している実行があるとその投入が skip され、停滞した実行が
// すでに遅らせた回復がもう 1 間隔分待つことになる。2 つの値は別パッケージにあるため、
// 両者を関係づけるものはこのテストしかない。
func TestReconcileExportsTimeoutFitsInterval(t *testing.T) {
	t.Parallel()

	if usecase.ReconcileExportsTimeout >= reconcileExportsInterval {
		t.Errorf("usecase.ReconcileExportsTimeout = %v, want < %v", usecase.ReconcileExportsTimeout, reconcileExportsInterval)
	}
}

// TestExportRecoveryWorkers_Timeout keeps both recovery runs bounded. Neither
// job's work is a fixed amount — reconciliation walks a backlog and the sweep
// lists every stored archive — so a run that stopped responding would otherwise
// hold its worker for as long as the process lives.
//
// [Ja] TestExportRecoveryWorkers_Timeout は 2 つの回復処理の実行が区切られていること
// を保つ。どちらの仕事量も固定ではなく (リコンシリエーションはバックログを走査し、
// 掃除は保存済みの全アーカイブを一覧する)、応答しなくなった実行は、これが無いと
// プロセスが生きている限り worker を占有する。
func TestExportRecoveryWorkers_Timeout(t *testing.T) {
	t.Parallel()

	if got := NewReconcileExportsWorker(nil).Timeout(nil); got != usecase.ReconcileExportsTimeout {
		t.Errorf("ReconcileExportsWorker.Timeout() = %v, want %v", got, usecase.ReconcileExportsTimeout)
	}
	if got := NewCleanupOrphanExportObjectsWorker(nil).Timeout(nil); got != usecase.CleanupOrphanExportObjectsTimeout {
		t.Errorf("CleanupOrphanExportObjectsWorker.Timeout() = %v, want %v", got, usecase.CleanupOrphanExportObjectsTimeout)
	}
}

// TestCleanupOldExportsWorker_Timeout keeps the cleanup bounded. It runs on the
// default queue alongside the timeline delivery and the emails, so a run
// waiting on a storage that stopped answering would hold one of those workers
// for as long as the process lives.
//
// [Ja] TestCleanupOldExportsWorker_Timeout は掃除が区切られていることを保つ。この
// 掃除はタイムライン配信やメール送信と同じ既定キューで動くため、応答しなくなった
// ストレージを待つ実行は、プロセスが生きている限りそれらの worker の 1 つを占有する。
func TestCleanupOldExportsWorker_Timeout(t *testing.T) {
	t.Parallel()

	if got := NewCleanupOldExportsWorker(nil).Timeout(nil); got != usecase.CleanupOldExportsTimeout {
		t.Errorf("CleanupOldExportsWorker.Timeout() = %v, want %v", got, usecase.CleanupOldExportsTimeout)
	}
}

// TestCleanupOldExportsWorker_Work pins the Worker boundary: valid job input
// is converted to the domain ID, UseCase failures reach River, and malformed
// input never calls the UseCase.
//
// [Ja] TestCleanupOldExportsWorker_Work は Worker 境界を固定する。有効なジョブ入力は
// domain ID へ変換し、UseCase の失敗は River へ返し、不正入力では UseCase を呼ばない。
func TestCleanupOldExportsWorker_Work(t *testing.T) {
	t.Parallel()

	const profileID = "018f2f4b-8f98-7e28-92d8-ffaf00371c91"
	job := func(profileID string) *river.Job[dispatcher.CleanupOldExportsArgs] {
		return &river.Job[dispatcher.CleanupOldExportsArgs]{
			JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 5},
			Args:   dispatcher.CleanupOldExportsArgs{ProfileID: profileID},
		}
	}

	t.Run("有効な profile_id を domain ID に変換して 1 回実行する", func(t *testing.T) {
		t.Parallel()

		executor := &recordingCleanupOldExportsExecutor{}
		err := NewCleanupOldExportsWorker(executor).Work(context.Background(), job(profileID))
		if err != nil {
			t.Fatalf("Work() error = %v", err)
		}
		if len(executor.profileIDs) != 1 {
			t.Fatalf("Execute() call count = %d, want 1", len(executor.profileIDs))
		}
		if got := executor.profileIDs[0].String(); got != profileID {
			t.Errorf("Execute() profileID = %s, want %s", got, profileID)
		}
	})

	t.Run("UseCase のエラーを River へ返す", func(t *testing.T) {
		t.Parallel()

		executeErr := errors.New("injected cleanup failure")
		executor := &recordingCleanupOldExportsExecutor{err: executeErr}
		err := NewCleanupOldExportsWorker(executor).Work(context.Background(), job(profileID))
		if !errors.Is(err, executeErr) {
			t.Errorf("Work() error = %v, want %v", err, executeErr)
		}
		if len(executor.profileIDs) != 1 {
			t.Errorf("Execute() call count = %d, want 1", len(executor.profileIDs))
		}
	})

	t.Run("不正な profile_id は UseCase を呼ばずエラーにする", func(t *testing.T) {
		t.Parallel()

		executor := &recordingCleanupOldExportsExecutor{}
		err := NewCleanupOldExportsWorker(executor).Work(context.Background(), job("not-a-uuid"))
		if err == nil {
			t.Fatal("Work() = nil, want an error")
		}
		if len(executor.profileIDs) != 0 {
			t.Errorf("Execute() call count = %d, want 0", len(executor.profileIDs))
		}
	})
}

// TestSendExportCompletedEmailWorker_Timeout keeps the delivery bounded. It
// runs on the default queue alongside the timeline delivery, so an attempt
// waiting on a mail provider that stopped answering would hold one of those
// workers for as long as the process lives.
//
// [Ja] TestSendExportCompletedEmailWorker_Timeout は配信が区切られていることを保つ。
// この配信はタイムライン配信と同じ既定キューで動くため、応答しなくなったメール
// プロバイダーを待つ試行は、プロセスが生きている限りそれらの worker の 1 つを占有する。
func TestSendExportCompletedEmailWorker_Timeout(t *testing.T) {
	t.Parallel()

	if got := NewSendExportCompletedEmailWorker(nil).Timeout(nil); got != usecase.SendExportCompletedEmailTimeout {
		t.Errorf("SendExportCompletedEmailWorker.Timeout() = %v, want %v", got, usecase.SendExportCompletedEmailTimeout)
	}
}

// TestSendExportCompletedEmailWorker_Work pins the Worker boundary: valid job
// input is converted to the domain ID, UseCase failures reach River so the
// delivery is retried, and malformed input never calls the UseCase.
//
// [Ja] TestSendExportCompletedEmailWorker_Work は Worker 境界を固定する。有効なジョブ
// 入力は domain ID へ変換し、UseCase の失敗は配信が再試行されるよう River へ返し、
// 不正入力では UseCase を呼ばない。
func TestSendExportCompletedEmailWorker_Work(t *testing.T) {
	t.Parallel()

	const exportID = "018f2f4b-8f98-7e28-92d8-ffaf00371c91"
	job := func(exportID string) *river.Job[dispatcher.SendExportCompletedEmailArgs] {
		return &river.Job[dispatcher.SendExportCompletedEmailArgs]{
			JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 5},
			Args:   dispatcher.SendExportCompletedEmailArgs{ExportID: exportID},
		}
	}

	t.Run("有効な export_id を domain ID に変換して 1 回実行する", func(t *testing.T) {
		t.Parallel()

		executor := &recordingSendExportCompletedEmailExecutor{}
		err := NewSendExportCompletedEmailWorker(executor).Work(context.Background(), job(exportID))
		if err != nil {
			t.Fatalf("Work() error = %v", err)
		}
		if len(executor.exportIDs) != 1 {
			t.Fatalf("Execute() call count = %d, want 1", len(executor.exportIDs))
		}
		if got := executor.exportIDs[0].String(); got != exportID {
			t.Errorf("Execute() exportID = %s, want %s", got, exportID)
		}
	})

	t.Run("UseCase のエラーを River へ返す", func(t *testing.T) {
		t.Parallel()

		executeErr := errors.New("injected delivery failure")
		executor := &recordingSendExportCompletedEmailExecutor{err: executeErr}
		err := NewSendExportCompletedEmailWorker(executor).Work(context.Background(), job(exportID))
		if !errors.Is(err, executeErr) {
			t.Errorf("Work() error = %v, want %v", err, executeErr)
		}
		if len(executor.exportIDs) != 1 {
			t.Errorf("Execute() call count = %d, want 1", len(executor.exportIDs))
		}
	})

	t.Run("不正な export_id は UseCase を呼ばずエラーにする", func(t *testing.T) {
		t.Parallel()

		executor := &recordingSendExportCompletedEmailExecutor{}
		err := NewSendExportCompletedEmailWorker(executor).Work(context.Background(), job("not-a-uuid"))
		if err == nil {
			t.Fatal("Work() = nil, want an error")
		}
		if len(executor.exportIDs) != 0 {
			t.Errorf("Execute() call count = %d, want 0", len(executor.exportIDs))
		}
	})
}

// TestGenerateExportWorker_Timeout keeps the attempt bounded: without a timeout
// a stalled generation would hold the single-worker export queue for as long as
// the process lives, and no other profile's export could start.
//
// [Ja] TestGenerateExportWorker_Timeout は試行が区切られていることを保つ。timeout が
// 無いと、停止した生成が worker 1 つの export キューをプロセスが生きている限り占有し、
// 他プロフィールのエクスポートを開始できなくなる。
func TestGenerateExportWorker_Timeout(t *testing.T) {
	t.Parallel()

	got := NewGenerateExportWorker(nil).Timeout(nil)
	if got != usecase.GenerateExportTimeout {
		t.Errorf("Timeout() = %v, want %v", got, usecase.GenerateExportTimeout)
	}
}

func TestGenerateExportWorker_Work(t *testing.T) {
	t.Parallel()

	const exportID = "018f2f4b-8f98-7e28-92d8-ffaf00371c91"
	tests := []struct {
		name             string
		attempt          int
		maxAttempts      int
		wantFinalAttempt bool
	}{
		{
			name:             "最終試行前",
			attempt:          2,
			maxAttempts:      3,
			wantFinalAttempt: false,
		},
		{
			name:             "最終試行",
			attempt:          3,
			maxAttempts:      3,
			wantFinalAttempt: true,
		},
		{
			name:             "最大試行回数を超えた再実行",
			attempt:          4,
			maxAttempts:      3,
			wantFinalAttempt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor := &recordingGenerateExportExecutor{}
			worker := NewGenerateExportWorker(executor)
			err := worker.Work(context.Background(), &river.Job[dispatcher.GenerateExportArgs]{
				JobRow: &rivertype.JobRow{
					Attempt:     tt.attempt,
					MaxAttempts: tt.maxAttempts,
				},
				Args: dispatcher.GenerateExportArgs{
					ExportID: exportID,
				},
			})
			if err != nil {
				t.Fatalf("Work() error = %v", err)
			}
			if len(executor.inputs) != 1 {
				t.Fatalf("Execute() call count = %d, want 1", len(executor.inputs))
			}

			got := executor.inputs[0]
			if got.ExportID.String() != exportID {
				t.Errorf("got.ExportID = %s, want %s", got.ExportID.String(), exportID)
			}
			if got.IsFinalAttempt != tt.wantFinalAttempt {
				t.Errorf("got.IsFinalAttempt = %v, want %v", got.IsFinalAttempt, tt.wantFinalAttempt)
			}
		})
	}

	t.Run("不正な export_id は UseCase を呼ばずエラーにする", func(t *testing.T) {
		t.Parallel()

		executor := &recordingGenerateExportExecutor{}
		worker := NewGenerateExportWorker(executor)
		err := worker.Work(context.Background(), &river.Job[dispatcher.GenerateExportArgs]{
			JobRow: &rivertype.JobRow{
				Attempt:     1,
				MaxAttempts: 3,
			},
			Args: dispatcher.GenerateExportArgs{
				ExportID: "not-a-uuid",
			},
		})
		if err == nil {
			t.Fatal("Work() = nil, want an error")
		}
		if len(executor.inputs) != 0 {
			t.Errorf("Execute() call count = %d, want 0", len(executor.inputs))
		}
	})
}
