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

	// MaxWorkersはRiverクライアント単位の設定であり、ここでは生成を
	// 1プロセスにつき1本に制限している。生成はDBからオブジェクトストレージへ
	// アーカイブ全体をストリーミングするため、同時実行すると1プロセスに必要な
	// メモリと帯域がその数だけ増える。
	exportQueue, ok := configs[dispatcher.QueueExport]
	if !ok {
		t.Fatalf("%qキューが登録されていません", dispatcher.QueueExport)
	}
	if exportQueue.MaxWorkers != 1 {
		t.Errorf("%qキューのMaxWorkers = %d、期待値 = 1", dispatcher.QueueExport, exportQueue.MaxWorkers)
	}

	// exportキューは追加であって置き換えではない。既定キューは引き続き
	// タイムライン配信とメール送信を処理する。
	defaultQueue, ok := configs[river.QueueDefault]
	if !ok {
		t.Fatalf("%qキューが登録されていません", river.QueueDefault)
	}
	if defaultQueue.MaxWorkers != 10 {
		t.Errorf("%qキューのMaxWorkers = %d、期待値 = 10", river.QueueDefault, defaultQueue.MaxWorkers)
	}
}

// jobArgsWithInsertOptsはTestJobQueuesAreServedが対象にするArgs型が
// 満たすべきinterface。Kindはサブテスト名に、InsertOptsはキュー名に使う。この型で
// テーブルを宣言することで、InsertOptsを持たないArgs型はテスト実行時ではなく
// コンパイル時に検出される。
type jobArgsWithInsertOpts interface {
	river.JobArgs
	river.JobArgsWithInsertOpts
}

// TestJobQueuesAreServedは、本workerがpollingしないキューへジョブが投入される
// 事態を防ぐ。Riverはその投入を受け付けてしまうため、ジョブは何の兆候もなくavailable
// のまま残り続ける。対象はdispatcherが定義する全Args型とし、エクスポート系に限らない。
// workerが複数のキューを処理するようになった以上、どのジョブでもqueueConfigsが
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
				t.Errorf("%qキューがWorkerクライアントに登録されていません", queue)
			}
		})
	}
}

// configuredExportUsecasesは、オブジェクトストレージが設定されたデプロイを
// 表すエクスポート系の束を返す。これらのテストはWorkerが登録されたかどうかを観測する
// だけで実行はしないため、UseCaseは空の値でよい。
func configuredExportUsecases() ExportUsecases {
	return ExportUsecases{
		Generate:             &usecase.GenerateExportUsecase{},
		CleanupOld:           &usecase.CleanupOldExportsUsecase{},
		SendCompletedEmail:   &usecase.SendExportCompletedEmailUsecase{},
		Reconcile:            &usecase.ReconcileExportsUsecase{},
		CleanupOrphanObjects: &usecase.CleanupOrphanExportObjectsUsecase{},
	}
}

// TestRegisterWorkers_Exportはオブジェクトストレージによるゲートを固定する。
// ストレージが未設定ならアーカイブのアップロード先が無く、そもそもエクスポート行を
// 作成できないため、エクスポート系Workerはすべて登録せず、プロセスは他のジョブを
// 引き続き処理できる必要がある。
//
// 登録の有無は同じWorkerを再登録して観測する。Riverは1つのジョブ種別に2つ目の
// Workerを拒否するため、エラーがnilならその種別は空いており、エラーが返れば既に
// 使われている。
func TestRegisterWorkers_Export(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		exportUCs      ExportUsecases
		wantRegistered bool
	}{
		{
			name:           "オブジェクトストレージ未設定ならエクスポート系Workerを登録しない",
			exportUCs:      ExportUsecases{},
			wantRegistered: false,
		},
		{
			name:           "オブジェクトストレージ設定済みならエクスポート系Workerを登録する",
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
					t.Errorf("%sの登録 = %v、期待値 = %v (AddWorkerSafely()のエラー = %v)", name, registered, tt.wantRegistered, err)
				}
			}
		})
	}
}

// TestExportPeriodicJobSpecsはエクスポートの回復処理を駆動するスケジュールを
// 固定する。この2つのジョブを投入するものは他に無いため、スケジュールが欠けると回復
// 処理が黙って止まる。ジョブの投入が失われたエクスポートはqueuedのまま残り、孤児
// オブジェクトは永久に課金され続けることになる。
func TestExportPeriodicJobSpecs(t *testing.T) {
	t.Parallel()

	t.Run("オブジェクトストレージ設定済みなら2つのスケジュールを登録する", func(t *testing.T) {
		t.Parallel()

		got := exportPeriodicJobSpecs(configuredExportUsecases())

		want := []periodicJobSpec{
			{args: dispatcher.ReconcileExportsArgs{}, interval: 5 * time.Minute, runOnStart: true},
			{args: dispatcher.CleanupOrphanExportObjectsArgs{}, interval: 24 * time.Hour, runOnStart: true},
		}
		if len(got) != len(want) {
			t.Fatalf("スケジュールの件数 = %d、期待値 = %d", len(got), len(want))
		}
		for i, wantSpec := range want {
			if got[i] != wantSpec {
				t.Errorf("スケジュール[%d] = %+v、期待値 = %+v", i, got[i], wantSpec)
			}
		}
	})

	// Workerを登録していないジョブをスケジュールすると、間隔ごとに投入され、
	// 処理するものが無いままavailableで残り続ける。そのためスケジュールは登録と
	// 同じゲートに従う。
	t.Run("オブジェクトストレージ未設定ならスケジュールを登録しない", func(t *testing.T) {
		t.Parallel()

		if got := exportPeriodicJobSpecs(ExportUsecases{}); len(got) != 0 {
			t.Errorf("スケジュール = %+v、期待値 = なし", got)
		}
	})

	t.Run("Riverの定期ジョブへ変換する", func(t *testing.T) {
		t.Parallel()

		specs := exportPeriodicJobSpecs(configuredExportUsecases())
		if got := toPeriodicJobs(specs); len(got) != len(specs) {
			t.Fatalf("定期ジョブの件数 = %d、期待値 = %d", len(got), len(specs))
		}

		// river.PeriodicJobはコンストラクタを非公開フィールドに持つため、
		// スケジュールが何を投入するかはコンストラクタ自体を通して読み戻す。
		wantKinds := []string{"reconcile_exports", "cleanup_orphan_export_objects"}
		if len(specs) != len(wantKinds) {
			t.Fatalf("スケジュールの件数 = %d、期待値 = %d", len(specs), len(wantKinds))
		}
		for i, spec := range specs {
			args, opts := periodicJobConstructor(spec)()
			if got := args.Kind(); got != wantKinds[i] {
				t.Errorf("コンストラクタ[%d] のKind = %q、期待値 = %q", i, got, wantKinds[i])
			}
			// オプションを返さないことにより、投入はArgs型が宣言するオプションを
			// 使う。ここで何らかのオプションを返すと、空の値であっても、同じ時間枠で
			// 済んだ処理を起動時投入が繰り返さないための一意性が失われる。
			if opts != nil {
				t.Errorf("コンストラクタ[%d] のopts = %+v、期待値 = nil", i, opts)
			}
		}
	})
}

// TestReconcileExportsTimeoutFitsIntervalは、リコンシリエーションのtimeoutを
// 定期実行間隔より小さく保つ。このジョブはrunningを含む状態で一意なため、次の定期投入
// の時点でまだworkerを保持している実行があるとその投入がskipされ、停滞した実行が
// すでに遅らせた回復がもう1間隔分待つことになる。2つの値は別パッケージにあるため、
// 両者を関係づけるものはこのテストしかない。
func TestReconcileExportsTimeoutFitsInterval(t *testing.T) {
	t.Parallel()

	if usecase.ReconcileExportsTimeout >= reconcileExportsInterval {
		t.Errorf("usecase.ReconcileExportsTimeout = %v、期待値 = %v未満", usecase.ReconcileExportsTimeout, reconcileExportsInterval)
	}
}

// TestExportRecoveryWorkers_Timeoutは2つの回復処理の実行が区切られていること
// を保つ。どちらの仕事量も固定ではなく (リコンシリエーションはバックログを走査し、
// 掃除は保存済みの全アーカイブを一覧する)、応答しなくなった実行は、これが無いと
// プロセスが生きている限りworkerを占有する。
func TestExportRecoveryWorkers_Timeout(t *testing.T) {
	t.Parallel()

	if got := NewReconcileExportsWorker(nil).Timeout(nil); got != usecase.ReconcileExportsTimeout {
		t.Errorf("ReconcileExportsWorker.Timeout() = %v、期待値 = %v", got, usecase.ReconcileExportsTimeout)
	}
	if got := NewCleanupOrphanExportObjectsWorker(nil).Timeout(nil); got != usecase.CleanupOrphanExportObjectsTimeout {
		t.Errorf("CleanupOrphanExportObjectsWorker.Timeout() = %v、期待値 = %v", got, usecase.CleanupOrphanExportObjectsTimeout)
	}
}

// TestCleanupOldExportsWorker_Timeoutは掃除が区切られていることを保つ。この
// 掃除はタイムライン配信やメール送信と同じ既定キューで動くため、応答しなくなった
// ストレージを待つ実行は、プロセスが生きている限りそれらのworkerの1つを占有する。
func TestCleanupOldExportsWorker_Timeout(t *testing.T) {
	t.Parallel()

	if got := NewCleanupOldExportsWorker(nil).Timeout(nil); got != usecase.CleanupOldExportsTimeout {
		t.Errorf("CleanupOldExportsWorker.Timeout() = %v、期待値 = %v", got, usecase.CleanupOldExportsTimeout)
	}
}

// TestCleanupOldExportsWorker_WorkはWorker境界を固定する。有効なジョブ入力は
// domain IDへ変換し、UseCaseの失敗はRiverへ返し、不正入力ではUseCaseを呼ばない。
func TestCleanupOldExportsWorker_Work(t *testing.T) {
	t.Parallel()

	const profileID = "018f2f4b-8f98-7e28-92d8-ffaf00371c91"
	job := func(profileID string) *river.Job[dispatcher.CleanupOldExportsArgs] {
		return &river.Job[dispatcher.CleanupOldExportsArgs]{
			JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 5},
			Args:   dispatcher.CleanupOldExportsArgs{ProfileID: profileID},
		}
	}

	t.Run("有効なprofile_idをdomain IDに変換して1回実行する", func(t *testing.T) {
		t.Parallel()

		executor := &recordingCleanupOldExportsExecutor{}
		err := NewCleanupOldExportsWorker(executor).Work(context.Background(), job(profileID))
		if err != nil {
			t.Fatalf("Work()のエラー = %v", err)
		}
		if len(executor.profileIDs) != 1 {
			t.Fatalf("Execute()の呼び出し回数 = %d、期待値 = 1", len(executor.profileIDs))
		}
		if got := executor.profileIDs[0].String(); got != profileID {
			t.Errorf("Execute()のprofileID = %s、期待値 = %s", got, profileID)
		}
	})

	t.Run("UseCaseのエラーをRiverへ返す", func(t *testing.T) {
		t.Parallel()

		executeErr := errors.New("injected cleanup failure")
		executor := &recordingCleanupOldExportsExecutor{err: executeErr}
		err := NewCleanupOldExportsWorker(executor).Work(context.Background(), job(profileID))
		if !errors.Is(err, executeErr) {
			t.Errorf("Work()のエラー = %v、期待値 = %v", err, executeErr)
		}
		if len(executor.profileIDs) != 1 {
			t.Errorf("Execute()の呼び出し回数 = %d、期待値 = 1", len(executor.profileIDs))
		}
	})

	t.Run("不正なprofile_idはUseCaseを呼ばずエラーにする", func(t *testing.T) {
		t.Parallel()

		executor := &recordingCleanupOldExportsExecutor{}
		err := NewCleanupOldExportsWorker(executor).Work(context.Background(), job("not-a-uuid"))
		if err == nil {
			t.Fatal("Work()のエラー = nil、エラーを期待")
		}
		if len(executor.profileIDs) != 0 {
			t.Errorf("Execute()の呼び出し回数 = %d、期待値 = 0", len(executor.profileIDs))
		}
	})
}

// TestSendExportCompletedEmailWorker_Timeoutは配信が区切られていることを保つ。
// この配信はタイムライン配信と同じ既定キューで動くため、応答しなくなったメール
// プロバイダーを待つ試行は、プロセスが生きている限りそれらのworkerの1つを占有する。
func TestSendExportCompletedEmailWorker_Timeout(t *testing.T) {
	t.Parallel()

	if got := NewSendExportCompletedEmailWorker(nil).Timeout(nil); got != usecase.SendExportCompletedEmailTimeout {
		t.Errorf("SendExportCompletedEmailWorker.Timeout() = %v、期待値 = %v", got, usecase.SendExportCompletedEmailTimeout)
	}
}

// TestSendExportCompletedEmailWorker_WorkはWorker境界を固定する。有効なジョブ
// 入力はdomain IDへ変換し、UseCaseの失敗は配信が再試行されるようRiverへ返し、
// 不正入力ではUseCaseを呼ばない。
func TestSendExportCompletedEmailWorker_Work(t *testing.T) {
	t.Parallel()

	const exportID = "018f2f4b-8f98-7e28-92d8-ffaf00371c91"
	job := func(exportID string) *river.Job[dispatcher.SendExportCompletedEmailArgs] {
		return &river.Job[dispatcher.SendExportCompletedEmailArgs]{
			JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 5},
			Args:   dispatcher.SendExportCompletedEmailArgs{ExportID: exportID},
		}
	}

	t.Run("有効なexport_idをdomain IDに変換して1回実行する", func(t *testing.T) {
		t.Parallel()

		executor := &recordingSendExportCompletedEmailExecutor{}
		err := NewSendExportCompletedEmailWorker(executor).Work(context.Background(), job(exportID))
		if err != nil {
			t.Fatalf("Work()のエラー = %v", err)
		}
		if len(executor.exportIDs) != 1 {
			t.Fatalf("Execute()の呼び出し回数 = %d、期待値 = 1", len(executor.exportIDs))
		}
		if got := executor.exportIDs[0].String(); got != exportID {
			t.Errorf("Execute()のexportID = %s、期待値 = %s", got, exportID)
		}
	})

	t.Run("UseCaseのエラーをRiverへ返す", func(t *testing.T) {
		t.Parallel()

		executeErr := errors.New("injected delivery failure")
		executor := &recordingSendExportCompletedEmailExecutor{err: executeErr}
		err := NewSendExportCompletedEmailWorker(executor).Work(context.Background(), job(exportID))
		if !errors.Is(err, executeErr) {
			t.Errorf("Work()のエラー = %v、期待値 = %v", err, executeErr)
		}
		if len(executor.exportIDs) != 1 {
			t.Errorf("Execute()の呼び出し回数 = %d、期待値 = 1", len(executor.exportIDs))
		}
	})

	t.Run("不正なexport_idはUseCaseを呼ばずエラーにする", func(t *testing.T) {
		t.Parallel()

		executor := &recordingSendExportCompletedEmailExecutor{}
		err := NewSendExportCompletedEmailWorker(executor).Work(context.Background(), job("not-a-uuid"))
		if err == nil {
			t.Fatal("Work()のエラー = nil、エラーを期待")
		}
		if len(executor.exportIDs) != 0 {
			t.Errorf("Execute()の呼び出し回数 = %d、期待値 = 0", len(executor.exportIDs))
		}
	})
}

// TestGenerateExportWorker_Timeoutは試行が区切られていることを保つ。timeoutが
// 無いと、停止した生成がworker 1つのexportキューをプロセスが生きている限り占有し、
// 他プロフィールのエクスポートを開始できなくなる。
func TestGenerateExportWorker_Timeout(t *testing.T) {
	t.Parallel()

	got := NewGenerateExportWorker(nil).Timeout(nil)
	if got != usecase.GenerateExportTimeout {
		t.Errorf("Timeout() = %v、期待値 = %v", got, usecase.GenerateExportTimeout)
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
				t.Fatalf("Work()のエラー = %v", err)
			}
			if len(executor.inputs) != 1 {
				t.Fatalf("Execute()の呼び出し回数 = %d、期待値 = 1", len(executor.inputs))
			}

			got := executor.inputs[0]
			if got.ExportID.String() != exportID {
				t.Errorf("got.ExportID = %s、期待値 = %s", got.ExportID.String(), exportID)
			}
			if got.IsFinalAttempt != tt.wantFinalAttempt {
				t.Errorf("got.IsFinalAttempt = %v、期待値 = %v", got.IsFinalAttempt, tt.wantFinalAttempt)
			}
		})
	}

	t.Run("不正なexport_idはUseCaseを呼ばずエラーにする", func(t *testing.T) {
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
			t.Fatal("Work()のエラー = nil、エラーを期待")
		}
		if len(executor.inputs) != 0 {
			t.Errorf("Execute()の呼び出し回数 = %d、期待値 = 0", len(executor.inputs))
		}
	})
}
