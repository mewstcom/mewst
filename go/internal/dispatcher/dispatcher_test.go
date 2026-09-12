package dispatcher

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// mockJobInserter はテスト用のモック
type mockJobInserter struct {
	called bool
	args   river.JobArgs
	opts   *river.InsertOpts
	err    error

	// uniqueSkipped makes Insert answer the way River answers an insert made
	// while a job for the same work intent is still outstanding: no error, and a
	// result flagged as skipped. Reconciliation counts only the candidates it
	// actually queues, so the Enqueue methods have to pass this through instead
	// of reporting it as a successful insert.
	//
	// [Ja] uniqueSkipped は、同じ作業依頼のジョブが未完了のまま残っているときの投入に
	// 対して River が返すのと同じ応答 (エラーではなく skip 済みの結果) を Insert に
	// させる。リコンシリエーションは実際にキューへ積んだ候補だけを数えるため、
	// Enqueue メソッドはこれを投入成功として扱わず呼び出し側へ伝える必要がある。
	uniqueSkipped bool
}

func (m *mockJobInserter) Insert(_ context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	m.called = true
	m.args = args
	m.opts = opts
	if m.err != nil {
		return nil, m.err
	}
	return &rivertype.JobInsertResult{UniqueSkippedAsDuplicate: m.uniqueSkipped}, nil
}

func TestEnqueueEmailConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		err := d.EnqueueEmailConfirmation(context.Background(), "test@example.com", "123456", "ja")
		if err != nil {
			t.Fatalf("EnqueueEmailConfirmation() error = %v", err)
		}

		if !mock.called {
			t.Fatal("Insert が呼ばれていません")
		}

		args, ok := mock.args.(SendEmailConfirmationArgs)
		if !ok {
			t.Fatalf("args の型が SendEmailConfirmationArgs ではありません: %T", mock.args)
		}
		if args.Email != "test@example.com" {
			t.Errorf("Email = %s, want test@example.com", args.Email)
		}
		if args.Code != "123456" {
			t.Errorf("Code = %s, want 123456", args.Code)
		}
		if args.Locale != "ja" {
			t.Errorf("Locale = %s, want ja", args.Locale)
		}
		if mock.opts == nil {
			t.Fatal("InsertOpts が nil です")
		}
		if mock.opts.MaxAttempts != 5 {
			t.Errorf("MaxAttempts = %d, want 5", mock.opts.MaxAttempts)
		}
	})

	t.Run("異常系: inserter がエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		mock := &mockJobInserter{err: insertErr}
		d := NewDispatcher(mock)

		err := d.EnqueueEmailConfirmation(context.Background(), "test@example.com", "123456", "ja")
		if !errors.Is(err, insertErr) {
			t.Errorf("EnqueueEmailConfirmation() error = %v, want %v", err, insertErr)
		}
	})
}

func TestEnqueueFanoutPost(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		if err := d.EnqueueFanoutPost(context.Background(), "post-123"); err != nil {
			t.Fatalf("EnqueueFanoutPost() error = %v", err)
		}

		args, ok := mock.args.(FanoutPostArgs)
		if !ok {
			t.Fatalf("args の型が FanoutPostArgs ではありません: %T", mock.args)
		}
		if args.PostID != "post-123" {
			t.Errorf("PostID = %s, want post-123", args.PostID)
		}
		if mock.opts == nil || mock.opts.MaxAttempts != 5 {
			t.Errorf("InsertOpts が反映されていません: %+v", mock.opts)
		}
	})

	t.Run("異常系: inserter がエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		mock := &mockJobInserter{err: insertErr}
		d := NewDispatcher(mock)

		if err := d.EnqueueFanoutPost(context.Background(), "post-123"); !errors.Is(err, insertErr) {
			t.Errorf("EnqueueFanoutPost() error = %v, want %v", err, insertErr)
		}
	})
}

func TestEnqueueAddPostToTimeline(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock)

	if err := d.EnqueueAddPostToTimeline(context.Background(), "profile-123", "post-456"); err != nil {
		t.Fatalf("EnqueueAddPostToTimeline() error = %v", err)
	}

	args, ok := mock.args.(AddPostToTimelineArgs)
	if !ok {
		t.Fatalf("args の型が AddPostToTimelineArgs ではありません: %T", mock.args)
	}
	if args.ProfileID != "profile-123" {
		t.Errorf("ProfileID = %s, want profile-123", args.ProfileID)
	}
	if args.PostID != "post-456" {
		t.Errorf("PostID = %s, want post-456", args.PostID)
	}
}

func TestFanoutPostArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (FanoutPostArgs{}).Kind(); got != "fanout_post" {
		t.Errorf("Kind() = %s, want fanout_post", got)
	}
}

func TestAddPostToTimelineArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (AddPostToTimelineArgs{}).Kind(); got != "add_post_to_timeline" {
		t.Errorf("Kind() = %s, want add_post_to_timeline", got)
	}
}

func TestEnqueueGenerateExport(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueGenerateExport(context.Background(), "export-123")
		if err != nil {
			t.Fatalf("EnqueueGenerateExport() error = %v", err)
		}
		if !inserted {
			t.Error("EnqueueGenerateExport() inserted = false, want true")
		}

		args, ok := mock.args.(GenerateExportArgs)
		if !ok {
			t.Fatalf("args の型が GenerateExportArgs ではありません: %T", mock.args)
		}
		if args.ExportID != "export-123" {
			t.Errorf("ExportID = %s, want export-123", args.ExportID)
		}
		assertExportInsertOpts(t, mock.opts, GenerateExportArgs{}.InsertOpts())
	})

	t.Run("一意性で skip された場合は false を返す", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{uniqueSkipped: true}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueGenerateExport(context.Background(), "export-123")
		if err != nil {
			t.Fatalf("EnqueueGenerateExport() error = %v", err)
		}
		if inserted {
			t.Error("EnqueueGenerateExport() inserted = true, want false")
		}
	})

	t.Run("異常系: inserter がエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		mock := &mockJobInserter{err: insertErr}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueGenerateExport(context.Background(), "export-123")
		if !errors.Is(err, insertErr) {
			t.Errorf("EnqueueGenerateExport() error = %v, want %v", err, insertErr)
		}
		if inserted {
			t.Error("EnqueueGenerateExport() inserted = true, want false")
		}
	})
}

func TestEnqueueCleanupOldExports(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueCleanupOldExports(context.Background(), "profile-123")
		if err != nil {
			t.Fatalf("EnqueueCleanupOldExports() error = %v", err)
		}
		if !inserted {
			t.Error("EnqueueCleanupOldExports() inserted = false, want true")
		}

		args, ok := mock.args.(CleanupOldExportsArgs)
		if !ok {
			t.Fatalf("args の型が CleanupOldExportsArgs ではありません: %T", mock.args)
		}
		if args.ProfileID != "profile-123" {
			t.Errorf("ProfileID = %s, want profile-123", args.ProfileID)
		}
		assertExportInsertOpts(t, mock.opts, CleanupOldExportsArgs{}.InsertOpts())
	})

	t.Run("一意性で skip された場合は false を返す", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{uniqueSkipped: true}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueCleanupOldExports(context.Background(), "profile-123")
		if err != nil {
			t.Fatalf("EnqueueCleanupOldExports() error = %v", err)
		}
		if inserted {
			t.Error("EnqueueCleanupOldExports() inserted = true, want false")
		}
	})

	t.Run("異常系: inserter がエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		mock := &mockJobInserter{err: insertErr}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueCleanupOldExports(context.Background(), "profile-123")
		if !errors.Is(err, insertErr) {
			t.Errorf("EnqueueCleanupOldExports() error = %v, want %v", err, insertErr)
		}
		if inserted {
			t.Error("EnqueueCleanupOldExports() inserted = true, want false")
		}
	})
}

func TestEnqueueSendExportCompletedEmail(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueSendExportCompletedEmail(context.Background(), "export-123")
		if err != nil {
			t.Fatalf("EnqueueSendExportCompletedEmail() error = %v", err)
		}
		if !inserted {
			t.Error("EnqueueSendExportCompletedEmail() inserted = false, want true")
		}

		args, ok := mock.args.(SendExportCompletedEmailArgs)
		if !ok {
			t.Fatalf("args の型が SendExportCompletedEmailArgs ではありません: %T", mock.args)
		}
		if args.ExportID != "export-123" {
			t.Errorf("ExportID = %s, want export-123", args.ExportID)
		}
		assertExportInsertOpts(t, mock.opts, SendExportCompletedEmailArgs{}.InsertOpts())
	})

	t.Run("一意性で skip された場合は false を返す", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{uniqueSkipped: true}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueSendExportCompletedEmail(context.Background(), "export-123")
		if err != nil {
			t.Fatalf("EnqueueSendExportCompletedEmail() error = %v", err)
		}
		if inserted {
			t.Error("EnqueueSendExportCompletedEmail() inserted = true, want false")
		}
	})

	t.Run("異常系: inserter がエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		mock := &mockJobInserter{err: insertErr}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueSendExportCompletedEmail(context.Background(), "export-123")
		if !errors.Is(err, insertErr) {
			t.Errorf("EnqueueSendExportCompletedEmail() error = %v, want %v", err, insertErr)
		}
		if inserted {
			t.Error("EnqueueSendExportCompletedEmail() inserted = true, want false")
		}
	})
}

func TestEnqueueReconcileExports(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueReconcileExports(context.Background())
		if err != nil {
			t.Fatalf("EnqueueReconcileExports() error = %v", err)
		}
		if !inserted {
			t.Error("EnqueueReconcileExports() inserted = false, want true")
		}

		if _, ok := mock.args.(ReconcileExportsArgs); !ok {
			t.Fatalf("args の型が ReconcileExportsArgs ではありません: %T", mock.args)
		}
		assertExportInsertOpts(t, mock.opts, ReconcileExportsArgs{}.InsertOpts())
	})

	t.Run("一意性で skip された場合は false を返す", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{uniqueSkipped: true}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueReconcileExports(context.Background())
		if err != nil {
			t.Fatalf("EnqueueReconcileExports() error = %v", err)
		}
		if inserted {
			t.Error("EnqueueReconcileExports() inserted = true, want false")
		}
	})

	t.Run("異常系: inserter がエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		mock := &mockJobInserter{err: insertErr}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueReconcileExports(context.Background())
		if !errors.Is(err, insertErr) {
			t.Errorf("EnqueueReconcileExports() error = %v, want %v", err, insertErr)
		}
		if inserted {
			t.Error("EnqueueReconcileExports() inserted = true, want false")
		}
	})
}

func TestEnqueueCleanupOrphanExportObjects(t *testing.T) {
	t.Parallel()

	// The resume position has to reach the job unchanged: it is the only thing
	// that stops the next walk from starting at the beginning of the prefix
	// again, which is the whole reason a bounded run hands it over.
	//
	// [Ja] 再開位置はジョブへそのまま届く必要がある。次の走査がプレフィックスの先頭から
	// やり直すのを止めるのはこれだけであり、有界な実行がこれを引き渡すのはまさにその
	// ためである。
	t.Run("正常系: 再開位置を持つジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		const startAfter = "exports/profile-id/export-id.zip"
		inserted, err := d.EnqueueCleanupOrphanExportObjects(context.Background(), startAfter)
		if err != nil {
			t.Fatalf("EnqueueCleanupOrphanExportObjects() error = %v", err)
		}
		if !inserted {
			t.Error("EnqueueCleanupOrphanExportObjects() inserted = false, want true")
		}

		args, ok := mock.args.(CleanupOrphanExportObjectsArgs)
		if !ok {
			t.Fatalf("args の型が CleanupOrphanExportObjectsArgs ではありません: %T", mock.args)
		}
		if args.StartAfter != startAfter {
			t.Errorf("args.StartAfter = %q, want %q", args.StartAfter, startAfter)
		}
		assertExportInsertOpts(t, mock.opts, CleanupOrphanExportObjectsArgs{}.InsertOpts())
	})

	// The daily schedule inserts the sweep with no resume position, which is what
	// makes its uniqueness window collapse repeated inserts of the first segment.
	//
	// [Ja] 日次スケジュールは再開位置を持たない形で掃除を投入する。最初の区間に対する
	// 投入の繰り返しを一意性の時間枠がまとめるのは、この形であることによる。
	t.Run("再開位置が空なら先頭からの走査として投入する", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		if _, err := d.EnqueueCleanupOrphanExportObjects(context.Background(), ""); err != nil {
			t.Fatalf("EnqueueCleanupOrphanExportObjects() error = %v", err)
		}

		args, ok := mock.args.(CleanupOrphanExportObjectsArgs)
		if !ok {
			t.Fatalf("args の型が CleanupOrphanExportObjectsArgs ではありません: %T", mock.args)
		}
		if args != (CleanupOrphanExportObjectsArgs{}) {
			t.Errorf("args = %+v, want the zero value", args)
		}
	})

	t.Run("一意性で skip された場合は false を返す", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{uniqueSkipped: true}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueCleanupOrphanExportObjects(context.Background(), "")
		if err != nil {
			t.Fatalf("EnqueueCleanupOrphanExportObjects() error = %v", err)
		}
		if inserted {
			t.Error("EnqueueCleanupOrphanExportObjects() inserted = true, want false")
		}
	})

	t.Run("異常系: inserter がエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		mock := &mockJobInserter{err: insertErr}
		d := NewDispatcher(mock)

		inserted, err := d.EnqueueCleanupOrphanExportObjects(context.Background(), "")
		if !errors.Is(err, insertErr) {
			t.Errorf("EnqueueCleanupOrphanExportObjects() error = %v, want %v", err, insertErr)
		}
		if inserted {
			t.Error("EnqueueCleanupOrphanExportObjects() inserted = true, want false")
		}
	})
}

// assertExportInsertOpts checks that the options reaching Insert are the ones
// the Args type defines. Passing nil (or dropped options) would silently lose
// the queue, the attempt limit and above all the uniqueness that lets Create
// and reconciliation insert the same work intent without running it twice.
//
// [Ja] assertExportInsertOpts は Insert に渡るオプションが Args 型の定義どおりで
// あることを検証する。nil を渡す (あるいはオプションを落とす) と、キュー・試行回数の
// 上限、そして何より Create とリコンシリエーションが同じ作業依頼を投入しても二重に
// 実行されないようにしている一意性が、黙って失われる。
func assertExportInsertOpts(t *testing.T, got *river.InsertOpts, want river.InsertOpts) {
	t.Helper()

	if got == nil {
		t.Fatal("InsertOpts が nil です")
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("Insert に渡された InsertOpts = %+v, want %+v", *got, want)
	}
}

func TestExportArgs_Kind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args river.JobArgs
		want string
	}{
		{args: GenerateExportArgs{}, want: "generate_export"},
		{args: CleanupOldExportsArgs{}, want: "cleanup_old_exports"},
		{args: SendExportCompletedEmailArgs{}, want: "send_export_completed_email"},
		{args: ReconcileExportsArgs{}, want: "reconcile_exports"},
		{args: CleanupOrphanExportObjectsArgs{}, want: "cleanup_orphan_export_objects"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.args.Kind(); got != tt.want {
				t.Errorf("Kind() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExportArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	// The work-intent set is River's default without completed: a job that
	// finished without converging its export row has to be re-insertable at
	// once, not only after the job cleaner removes it. Available, pending,
	// running and scheduled stay because River rejects a set that omits any of
	// them, and an insert error would leave the work intent unqueued. Retryable
	// stays because a job awaiting retry is still outstanding work that a
	// duplicate insert must be skipped for rather than run alongside.
	//
	// [Ja] 作業依頼の一意性を判定する集合は River の既定から completed を除いたもの。
	// エクスポート行を収束させないまま終わったジョブは、job cleaner に消された後では
	// なく、すぐに再投入できる必要がある。available / pending / running / scheduled は、
	// これらを欠いた集合を River が拒否し、投入エラーになると作業依頼が未投入のまま
	// 残るため入れる。retryable は、再試行待ちのジョブも未処理の作業であり、重複した
	// 投入を並走させず skip する必要があるため入れる。
	wantStates := []rivertype.JobState{
		rivertype.JobStateAvailable,
		rivertype.JobStatePending,
		rivertype.JobStateRetryable,
		rivertype.JobStateRunning,
		rivertype.JobStateScheduled,
	}
	// Export work intents can be inserted repeatedly: generate, cleanup and email
	// use immediate and reconciliation paths, while reconciliation itself is
	// inserted by its periodic schedule. Uniqueness by args collapses overlapping
	// outstanding jobs into a no-op instead of a second run, so every case below
	// sets ByArgs.
	//
	// [Ja] エクスポート系の作業依頼は繰り返し投入されうる。生成・削除・通知には即時投入と
	// リコンシリエーションからの投入経路があり、リコンシリエーション自体は定期実行から
	// 投入される。args 単位の一意性により、未完了の同じ作業依頼が重なったときは二重実行
	// せず no-op にするため、以下のすべてのケースが ByArgs を設定する。
	tests := []struct {
		name string
		args river.JobArgsWithInsertOpts
		want river.InsertOpts
	}{
		// Generation gets its own queue so that a long export cannot occupy the
		// default workers, and five attempts so a transient failure does not end it.
		//
		// [Ja] 生成は長いエクスポートが既定の worker を占有しないよう専用キューを使い、
		// 一時的な失敗で終わらないよう 5 回まで試行する。
		{
			name: "generate_export",
			args: GenerateExportArgs{},
			want: river.InsertOpts{
				Queue:       QueueExport,
				MaxAttempts: 5,
				UniqueOpts:  river.UniqueOpts{ByArgs: true, ByState: wantStates},
			},
		},
		{
			name: "cleanup_old_exports",
			args: CleanupOldExportsArgs{},
			want: river.InsertOpts{
				Queue:       river.QueueDefault,
				MaxAttempts: 5,
				Priority:    3,
				UniqueOpts:  river.UniqueOpts{ByArgs: true, ByState: wantStates},
			},
		},
		{
			name: "send_export_completed_email",
			args: SendExportCompletedEmailArgs{},
			want: river.InsertOpts{
				Queue:       river.QueueDefault,
				MaxAttempts: 5,
				Priority:    2,
				UniqueOpts:  river.UniqueOpts{ByArgs: true, ByState: wantStates},
			},
		},
		// The periodic schedule is the retry: retrying in place would keep a failed
		// run retryable, and a retryable job blocks the next periodic insert through
		// the uniqueness check.
		//
		// [Ja] 定期スケジュールそのものが再試行にあたる。その場で再試行すると失敗した
		// 実行が retryable のまま残り、一意性判定によって次の定期投入を塞ぐ。
		{
			name: "reconcile_exports",
			args: ReconcileExportsArgs{},
			want: river.InsertOpts{
				Queue:       river.QueueDefault,
				MaxAttempts: 1,
				Priority:    3,
				UniqueOpts:  river.UniqueOpts{ByArgs: true, ByState: wantStates},
			},
		},
		// The sweep retries in place instead, because its schedule is daily:
		// leaving a transient object storage failure to the next run would keep
		// orphans billed for another day, while River's backoff uses the attempts
		// up in minutes. Completed is included only for this job so repeated
		// run-on-start inserts in one daily window do not repeat a successful sweep.
		//
		// [Ja] 一方この掃除はその場で再試行する。スケジュールが日次のため、
		// オブジェクトストレージの一時的な失敗を次回の実行に委ねると孤児オブジェクトの
		// 課金がもう 1 日続く。River のバックオフでは試行が数分で尽きる。このジョブに
		// 限って completed も含め、同じ日次時間枠の起動時投入で成功済みの掃除を繰り返さない。
		{
			name: "cleanup_orphan_export_objects",
			args: CleanupOrphanExportObjectsArgs{},
			want: river.InsertOpts{
				Queue:       river.QueueDefault,
				MaxAttempts: 5,
				Priority:    3,
				UniqueOpts: river.UniqueOpts{
					ByArgs:   true,
					ByPeriod: 24 * time.Hour,
					ByState:  rivertype.UniqueOptsByStateDefault(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// The whole struct is compared so that an option nobody thought to assert
			// cannot appear unnoticed. ExcludeKind would drop the kind from the unique
			// key and make generate and email collide on the same export ID, and
			// ByPeriod would turn work-intent uniqueness from "a job is still
			// outstanding" into "a job was inserted recently". The orphan sweep is
			// intentionally the sole exception, and its complete expected struct
			// fixes the daily period and completed-inclusive states above.
			//
			// [Ja] 誰もアサートしていないオプションが黙って増えないよう、構造体全体を
			// 比較する。ExcludeKind は一意キーから kind を外すため、生成と通知が同じ
			// エクスポート ID で衝突する。ByPeriod は作業依頼の一意性を「未完了のジョブが
			// ある」から「最近投入された」へ変えてしまう。孤児回収だけは意図的な例外で、
			// 完全な期待構造体により日次の期間と completed を含む状態集合を上で固定する。
			if got := tt.args.InsertOpts(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InsertOpts() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestGenerateExportMaxAttempts_MatchesInsertOpts pins the exported constant to
// the retry budget the job queue actually enforces. Reconciliation closes an
// export as failed once its attempts are used up, and it reads that number from
// the constant, so a MaxAttempts changed only in InsertOpts would make it give
// up early or keep retrying an export the queue already abandoned.
//
// [Ja] TestGenerateExportMaxAttempts_MatchesInsertOpts は、exported な定数を
// ジョブキューが実際に適用する再試行の予算に固定する。リコンシリエーションは試行を
// 使い切ったエクスポートを failed として閉じる際にこの定数を読むため、InsertOpts 側
// だけ MaxAttempts を変えると、早すぎる打ち切りか、キューがすでに諦めた
// エクスポートを再試行し続けるかのどちらかになる。
func TestGenerateExportMaxAttempts_MatchesInsertOpts(t *testing.T) {
	t.Parallel()

	if got := (GenerateExportArgs{}).InsertOpts().MaxAttempts; got != GenerateExportMaxAttempts {
		t.Errorf("InsertOpts().MaxAttempts = %d, want %d", got, GenerateExportMaxAttempts)
	}
}

// TestExportJobUniqueStates_IsRiverDefaultWithoutCompleted pins the doc claim
// that the set is River's default without completed. A River release that adds
// a state to the default set has to fail here rather than silently narrowing
// the uniqueness of the export jobs, or — if the added state is also required —
// failing every export Insert at run time and leaving the work intents queued.
//
// [Ja] TestExportJobUniqueStates_IsRiverDefaultWithoutCompleted は「River の既定集合
// から completed を除いたもの」という doc コメントの主張を固定する。River の更新で
// 既定集合に状態が追加されたときは、エクスポート系ジョブの一意性が黙って狭まったり、
// 追加された状態が必須集合にも入っていた場合に実行時の Insert がすべて失敗して作業依頼が
// queued のまま残ったりする前に、ここで落ちる必要がある。
func TestExportJobUniqueStates_IsRiverDefaultWithoutCompleted(t *testing.T) {
	t.Parallel()

	want := slices.DeleteFunc(rivertype.UniqueOptsByStateDefault(), func(state rivertype.JobState) bool {
		return state == rivertype.JobStateCompleted
	})

	if got := exportJobUniqueStates(); !slices.Equal(got, want) {
		t.Errorf("exportJobUniqueStates() = %v, want %v (River の既定集合から completed を除いたもの)", got, want)
	}
}

func TestExportJobUniqueStates_ReturnsIndependentSlices(t *testing.T) {
	t.Parallel()

	// The states are handed to River inside InsertOpts, so a shared backing
	// array would let one job's options mutate another's.
	//
	// [Ja] 状態集合は InsertOpts の一部として River に渡るため、backing array を
	// 共有していると、あるジョブのオプションが別のジョブのものを書き換えうる。
	first := exportJobUniqueStates()
	second := exportJobUniqueStates()
	first[0] = rivertype.JobStateCompleted

	if second[0] == rivertype.JobStateCompleted {
		t.Error("exportJobUniqueStates() が呼び出し間でスライスを共有しています")
	}
}

func TestDeferredInserter_DelegatesAfterSetInserter(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	deferred := &DeferredInserter{}
	deferred.SetInserter(mock)

	// The Dispatcher built around the DeferredInserter must reach the wired
	// inserter once SetInserter has been called.
	// [Ja] DeferredInserter を包んだ Dispatcher は、SetInserter 後に注入済みの
	// inserter へ到達できなければならない。
	d := NewDispatcher(deferred)
	if err := d.EnqueueFanoutPost(context.Background(), "post-123"); err != nil {
		t.Fatalf("EnqueueFanoutPost() error = %v", err)
	}

	if !mock.called {
		t.Fatal("注入した inserter の Insert が呼ばれていません")
	}
}

func TestSendEmailConfirmationArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (SendEmailConfirmationArgs{}).Kind(); got != "send_email_confirmation" {
		t.Errorf("Kind() = %s, want send_email_confirmation", got)
	}
}

func TestSendEmailConfirmationArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	opts := (SendEmailConfirmationArgs{}).InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("InsertOpts().Queue = %v, want %v", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 5 {
		t.Errorf("InsertOpts().MaxAttempts = %v, want 5", opts.MaxAttempts)
	}
	// Email is demoted to Priority 2 so the timeline delivery jobs (kept at the
	// default top priority) are worked first when the pool is saturated.
	// [Ja] メールは Priority 2 に降格し、プール飽和時にタイムライン配信ジョブ (既定の
	// 最高優先度のまま) を先に処理させる。
	if opts.Priority != 2 {
		t.Errorf("InsertOpts().Priority = %v, want 2 (demoted below timeline delivery jobs)", opts.Priority)
	}
}

func TestFanoutPostArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	opts := (FanoutPostArgs{}).InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("InsertOpts().Queue = %v, want %v", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 5 {
		t.Errorf("InsertOpts().MaxAttempts = %v, want 5", opts.MaxAttempts)
	}
	// Priority is left unset (0), which River treats as the default top priority
	// (1). It must outrank the demoted email job so fanout is fetched first.
	// [Ja] Priority は未設定 (0) で、River はこれを既定の最高優先度 (1) として扱う。
	// 降格したメールジョブより上位であり、fanout が先に fetch される必要がある。
	if opts.Priority != 0 {
		t.Errorf("InsertOpts().Priority = %v, want 0 (default top priority)", opts.Priority)
	}
	if emailOpts := (SendEmailConfirmationArgs{}).InsertOpts(); opts.Priority >= emailOpts.Priority {
		t.Errorf("fanout priority (%d) must outrank email priority (%d)", opts.Priority, emailOpts.Priority)
	}
}

func TestAddPostToTimelineArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	opts := (AddPostToTimelineArgs{}).InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("InsertOpts().Queue = %v, want %v", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 5 {
		t.Errorf("InsertOpts().MaxAttempts = %v, want 5", opts.MaxAttempts)
	}
	// Priority is left unset (0 = default top priority), keeping per-follower
	// delivery ahead of the demoted email job (Priority 2).
	// [Ja] Priority は未設定 (0 = 既定の最高優先度) で、フォロワー単位の配信を降格した
	// メールジョブ (Priority 2) より上位に保つ。
	if opts.Priority != 0 {
		t.Errorf("InsertOpts().Priority = %v, want 0 (default top priority)", opts.Priority)
	}
	if emailOpts := (SendEmailConfirmationArgs{}).InsertOpts(); opts.Priority >= emailOpts.Priority {
		t.Errorf("add_post priority (%d) must outrank email priority (%d)", opts.Priority, emailOpts.Priority)
	}
}
