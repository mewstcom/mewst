package usecase_test

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/exportfile"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// fakeExportObjectStorage stands in for the object storage and refuses any key
// outside exports/{profile_id}/{export_id}.zip, so a generation that writes to
// a key the cleanup and orphan recovery cannot derive fails the test instead of
// leaving an object nothing will ever collect.
//
// [Ja] fakeExportObjectStorage はオブジェクトストレージの代役で、
// exports/{profile_id}/{export_id}.zip の外のキーをすべて拒否する。cleanup や
// 孤児回収が導出できないキーへ書く生成は、回収されないオブジェクトを残すのではなく
// テストの失敗になる。
type fakeExportObjectStorage struct {
	t         *testing.T
	profileID model.ProfileID
	exportID  model.ExportID

	// uploadHook replaces the default upload when set, so a test can fail the
	// upload, stop reading the archive halfway, or change the export's state
	// while it is in flight.
	//
	// [Ja] uploadHook を設定すると既定のアップロードを置き換える。テストは
	// アップロードを失敗させたり、アーカイブの読み取りを途中で止めたり、
	// アップロード中にエクスポートの状態を変えたりできる。
	uploadHook func(ctx context.Context, body io.Reader) error

	mu           sync.Mutex
	objects      map[string][]byte
	uploadedKeys []string
	deletedKeys  []string
}

func newFakeExportObjectStorage(t *testing.T, profileID model.ProfileID, exportID model.ExportID) *fakeExportObjectStorage {
	t.Helper()
	return &fakeExportObjectStorage{
		t:         t,
		profileID: profileID,
		exportID:  exportID,
		objects:   map[string][]byte{},
	}
}

func (f *fakeExportObjectStorage) Upload(ctx context.Context, key string, body io.Reader) error {
	f.assertKey(key)

	f.mu.Lock()
	f.uploadedKeys = append(f.uploadedKeys, key)
	f.mu.Unlock()

	if f.uploadHook != nil {
		return f.uploadHook(ctx, body)
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = data
	return nil
}

func (f *fakeExportObjectStorage) Download(_ context.Context, key string) (io.ReadCloser, int64, error) {
	f.assertKey(key)

	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[key]
	if !ok {
		return nil, 0, fmt.Errorf("オブジェクトが存在しない (key: %s)", key)
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}

func (f *fakeExportObjectStorage) Delete(_ context.Context, key string) error {
	f.assertKey(key)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletedKeys = append(f.deletedKeys, key)
	delete(f.objects, key)
	return nil
}

func (f *fakeExportObjectStorage) ListPrefix(_ context.Context, prefix, _ string, _ func(key string, lastModified time.Time) error) error {
	f.t.Errorf("生成処理は ListPrefix を呼ばないはず (prefix: %s)", prefix)
	return nil
}

// assertKey fails the test unless the key follows the export convention and
// names the export under test.
//
// [Ja] assertKey は、キーがエクスポートの規約に従い、かつテスト対象のエクスポートを
// 指していない場合にテストを失敗させる。
func (f *fakeExportObjectStorage) assertKey(key string) {
	f.t.Helper()

	profileID, exportID, err := usecase.ParseExportObjectKey(key)
	if err != nil {
		f.t.Errorf("規約外のオブジェクトキーを操作した (key: %s): %v", key, err)
		return
	}
	if profileID != f.profileID || exportID != f.exportID {
		f.t.Errorf("別のエクスポートのオブジェクトキーを操作した (key: %s)", key)
	}
}

func (f *fakeExportObjectStorage) object(key string) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[key]
	return data, ok
}

func (f *fakeExportObjectStorage) uploads() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.uploadedKeys)
}

func (f *fakeExportObjectStorage) deletes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.deletedKeys)
}

// generateExportFixture is one export ready to be generated, together with the
// repositories and the fake storage the UseCase runs against.
//
// [Ja] generateExportFixture は生成可能な状態のエクスポート 1 件と、UseCase が
// 使うリポジトリおよび fake ストレージ。
type generateExportFixture struct {
	export     *model.Export
	objectKey  string
	exportRepo *repository.ExportRepository
	postRepo   *repository.ExportPostRepository
	actorRepo  *repository.ActorRepository
	userRepo   *repository.UserRepository
	storage    *fakeExportObjectStorage
	inserter   *exportJobInserter
	uc         *usecase.GenerateExportUsecase
}

// faultInjectingDBTX fails the one query a test wants to fail and passes every
// other one to the real transaction. The query is selected by a substring of
// the statement, for which the `-- name:` header sqlc emits is the stable
// choice. Failing a single statement is what puts the generation in a state
// only a partial failure produces, such as an archive that is already uploaded
// under a row that could not be moved to succeeded.
//
// [Ja] faultInjectingDBTX は、テストが失敗させたい 1 つのクエリだけを失敗させ、
// それ以外は実トランザクションへ委譲する。対象は文の部分一致で選び、sqlc が出力する
// `-- name:` ヘッダーが安定した指定先になる。1 文だけを失敗させることで、部分的な
// 失敗でしか生じない状態 (アップロード済みのアーカイブに対して、行を succeeded へ
// 進められない、など) を作る。
type faultInjectingDBTX struct {
	query.DBTX
	execErrorFor     string
	queryRowErrorFor string

	// beforeQueryRow runs just before the query selected by beforeQueryRowFor is
	// executed. It is how a test moves the row in the window between the read
	// that chose it and the guarded update that would take it, which is the only
	// way that update's guard can be made to miss.
	//
	// [Ja] beforeQueryRow は beforeQueryRowFor で選んだクエリの実行直前に走る。
	// 行を選んだ読み取りと、その行を引き受けるガード付き更新との間の窓で行を動かす
	// ためのもので、その更新のガードを外させる唯一の方法である。
	beforeQueryRowFor string
	beforeQueryRow    func()
}

func (db *faultInjectingDBTX) ExecContext(ctx context.Context, statement string, args ...any) (sql.Result, error) {
	if db.execErrorFor != "" && strings.Contains(statement, db.execErrorFor) {
		return nil, errors.New("注入したデータベースエラー")
	}
	return db.DBTX.ExecContext(ctx, statement, args...)
}

// QueryRowContext substitutes a row the caller cannot scan, which surfaces as a
// retrieval error from the repository. A row is substituted rather than an
// error returned because sql.Row carries an error only from the query that
// built it, so a query has to be run either way.
//
// [Ja] QueryRowContext は呼び出し側が Scan できない行に差し替え、repository からは
// 取得エラーとして表面化させる。エラーを返すのではなく行を差し替えるのは、sql.Row が
// エラーを自身を生成したクエリからしか運ばず、どのみちクエリを実行する必要があるため。
func (db *faultInjectingDBTX) QueryRowContext(ctx context.Context, statement string, args ...any) *sql.Row {
	if db.beforeQueryRow != nil && db.beforeQueryRowFor != "" && strings.Contains(statement, db.beforeQueryRowFor) {
		db.beforeQueryRow()
	}
	if db.queryRowErrorFor != "" && strings.Contains(statement, db.queryRowErrorFor) {
		return db.DBTX.QueryRowContext(ctx, "SELECT NULL::text")
	}
	return db.DBTX.QueryRowContext(ctx, statement, args...)
}

type panickingExportArchiveBuilder struct{}

func (panickingExportArchiveBuilder) NewArchive(_ io.Writer, _ usecase.ExportArchive) usecase.ExportArchiveWriter {
	panic("注入したアーカイブ panic")
}

type blockingExportArchiveBuilder struct {
	writerExited chan struct{}
}

func (b *blockingExportArchiveBuilder) NewArchive(w io.Writer, _ usecase.ExportArchive) usecase.ExportArchiveWriter {
	return &blockingExportArchiveWriter{
		w:            w,
		writerExited: b.writerExited,
	}
}

type blockingExportArchiveWriter struct {
	w            io.Writer
	writerExited chan struct{}
	closeOnce    sync.Once
}

func (w *blockingExportArchiveWriter) WriteIndex(context.Context) error {
	_, err := io.WriteString(w.w, "注入したアーカイブ")
	return err
}

func (*blockingExportArchiveWriter) OpenMonth(context.Context, usecase.ExportArchiveMonth) (usecase.ExportArchiveMonthWriter, error) {
	return nil, errors.New("WriteIndex の終了後に OpenMonth が呼ばれた")
}

func (w *blockingExportArchiveWriter) Close() error {
	w.closeOnce.Do(func() { close(w.writerExited) })
	return nil
}

// newGenerateExportFixture creates a user with the given locale and time zone,
// posts published at the given instants, and the queued export that
// materialized them.
//
// [Ja] newGenerateExportFixture は指定したロケールとタイムゾーンのユーザー、指定した
// 時点に公開された投稿、およびそれらを固定化した queued のエクスポートを作成する。
func newGenerateExportFixture(t *testing.T, tx *sql.Tx, locale, timeZone string, publishedAts ...time.Time) *generateExportFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithLocale(locale).
		WithTimeZone(timeZone).
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	oauthApplicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()
	for i, publishedAt := range publishedAts {
		testutil.NewPostBuilder(t, tx).
			WithProfileID(profileID).
			WithOauthApplicationID(oauthApplicationID).
			WithContent(fmt.Sprintf("投稿 %d の本文", i)).
			WithPublishedAt(publishedAt).
			Build()
	}

	exportRepo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
	postRepo := repository.NewExportPostRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	export, err := exportRepo.Create(context.Background(), repository.CreateExportInput{
		ProfileID: profileID,
		ActorID:   actorID,
	})
	if err != nil {
		t.Fatalf("エクスポートの作成に失敗: %v", err)
	}

	storage := newFakeExportObjectStorage(t, profileID, export.ID)
	inserter := newExportJobInserter(t)
	return &generateExportFixture{
		export:     export,
		objectKey:  usecase.ExportObjectKey(profileID, export.ID),
		exportRepo: exportRepo,
		postRepo:   postRepo,
		actorRepo:  actorRepo,
		userRepo:   userRepo,
		storage:    storage,
		inserter:   inserter,
		uc: usecase.NewGenerateExportUsecase(
			exportRepo,
			postRepo,
			actorRepo,
			userRepo,
			allowingExportProfileDeletionGuard{},
			exportfile.NewBuilder(),
			storage,
			dispatcher.NewDispatcher(inserter),
		),
	}
}

// reload returns the export's current row.
//
// [Ja] reload はエクスポートの現在の行を返す。
func (f *generateExportFixture) reload(t *testing.T) *model.Export {
	t.Helper()

	export, err := f.exportRepo.FindByID(context.Background(), f.export.ID)
	if err != nil {
		t.Fatalf("エクスポートの再取得に失敗: %v", err)
	}
	if export == nil {
		t.Fatal("エクスポートが存在しない")
	}
	return export
}

// snapshotCount returns how many posts the export's request-time snapshot still
// holds.
//
// [Ja] snapshotCount はエクスポートの申請時 snapshot に残っている投稿の件数を返す。
func (f *generateExportFixture) snapshotCount(t *testing.T) int64 {
	t.Helper()

	months, err := f.postRepo.ListMonthsByExportID(context.Background(), repository.ListExportPostMonthsByExportIDInput{
		ExportID: f.export.ID,
		Location: time.UTC,
	})
	if err != nil {
		t.Fatalf("snapshot の取得に失敗: %v", err)
	}
	var total int64
	for _, month := range months {
		total += month.PostCount
	}
	return total
}

// zipEntryNames returns the names of the entries in the uploaded archive.
//
// [Ja] zipEntryNames はアップロードされたアーカイブに含まれるエントリ名を返す。
func zipEntryNames(t *testing.T, data []byte) []string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip の読み取りに失敗: %v", err)
	}
	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	return names
}

// zipEntryContent returns the content of one entry of the uploaded archive.
//
// [Ja] zipEntryContent はアップロードされたアーカイブの 1 エントリの内容を返す。
func zipEntryContent(t *testing.T, data []byte, name string) string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip の読み取りに失敗: %v", err)
	}
	file, err := reader.Open(name)
	if err != nil {
		t.Fatalf("zip エントリ %q の取得に失敗: %v", name, err)
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("zip エントリ %q の読み取りに失敗: %v", name, err)
	}
	return string(content)
}

// postContentPattern matches the body of one post in a month's entry. The
// archive's HTML contract is fixed by the parser-based tests of the package
// that renders it, so recovering the bodies in document order is all that is
// needed here.
//
// [Ja] postContentPattern は月のエントリに含まれるポスト 1 件の本文にマッチする。
// アーカイブの HTML 契約は、それを描画するパッケージのパーサーによるテストが
// 固定しているため、ここで必要なのはドキュメント順に本文を復元することだけである。
var postContentPattern = regexp.MustCompile(`<p class="e-content">([^<]*)</p>`)

// postContents returns the body of every post of a month's entry, in document
// order.
//
// [Ja] postContents は月のエントリに含まれる各ポストの本文を、ドキュメント順で
// 返す。
func postContents(entry string) []string {
	matches := postContentPattern.FindAllStringSubmatch(entry, -1)
	contents := make([]string, 0, len(matches))
	for _, match := range matches {
		contents = append(contents, match[1])
	}
	return contents
}

func TestGenerateExportUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("アーカイブをアップロードして succeeded へ遷移し snapshot を破棄する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
		)

		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusSucceeded)
		}
		if got.ObjectKey == nil || *got.ObjectKey != fixture.objectKey {
			t.Errorf("got.ObjectKey = %v, want %v", got.ObjectKey, fixture.objectKey)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d, want 1", got.AttemptCount)
		}

		// The snapshot only served the attempt, so it must not outlive it.
		//
		// [Ja] snapshot は試行のためだけに存在したため、試行より長く残ってはならない。
		if count := fixture.snapshotCount(t); count != 0 {
			t.Errorf("succeeded 後の snapshot 件数 = %d, want 0", count)
		}

		data, ok := fixture.storage.object(fixture.objectKey)
		if !ok {
			t.Fatalf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
		wantEntries := []string{"index.html", "posts/2026-06.html", "posts/2026-07.html"}
		if got := zipEntryNames(t, data); !slices.Equal(got, wantEntries) {
			t.Errorf("zip エントリ = %v, want %v", got, wantEntries)
		}
		if content := zipEntryContent(t, data, "posts/2026-07.html"); !bytes.Contains([]byte(content), []byte("投稿 1 の本文")) {
			t.Errorf("posts/2026-07.html に投稿本文が含まれていない: %s", content)
		}

		// The success takes the download over from whatever the profile had
		// before, so the storage those exports still occupy is asked to be
		// released right away rather than waiting for the reconciliation that
		// backs this insert up.
		//
		// [Ja] この成功はプロフィールがそれまで持っていたものからダウンロードを
		// 引き継ぐため、それらのエクスポートが占有し続けるストレージの解放を、この
		// 投入を裏で支えるリコンシリエーションを待たずに直ちに要求する。
		assertJobFor(t, fixture.inserter, "cleanup_old_exports", fixture.export.ProfileID.String())

		// The requester is waiting for the mail, so the notification the success
		// just created is enqueued right away rather than left to the
		// reconciliation that runs every few minutes.
		//
		// [Ja] 申請者はメールを待っているため、この成功が作成した通知は数分おきの
		// リコンシリエーションに任せず直ちに投入する。
		assertJobFor(t, fixture.inserter, "send_export_completed_email", fixture.export.ID.String())
	})

	t.Run("完了メールジョブの投入に失敗しても成功を取り消さない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)
		fixture.inserter.failKinds["send_export_completed_email"] = true

		// The notification is a durable row of its own, created by the same
		// statement as the success, so a lost insert is re-derived from the
		// outbox. Reporting a failure would instead retry an attempt that reads a
		// terminal row and returns before enqueueing anything.
		//
		// [Ja] 通知は成功と同じ文で作成される、それ自体が durable な行であるため、
		// 失われた投入は outbox から再導出される。失敗を報告すると、終端状態の行を
		// 読んで何も投入せずに戻る試行を再試行することになる。
		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusSucceeded)
		}
		if _, ok := fixture.storage.object(fixture.objectKey); !ok {
			t.Errorf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
		// The old exports still have to be released even when the mail job could
		// not be enqueued: the two side effects converge independently.
		//
		// [Ja] メールのジョブを投入できなくても、旧エクスポートの解放は行われる必要が
		// ある。2 つの副作用はそれぞれ独立に収束する。
		assertJobFor(t, fixture.inserter, "cleanup_old_exports", fixture.export.ProfileID.String())
	})

	t.Run("旧エクスポート削除ジョブの投入に失敗しても成功を取り消さない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)
		fixture.inserter.failKinds["cleanup_old_exports"] = true

		// Reporting a failure here would retry an attempt with nothing left to do:
		// the retry reads a terminal row and returns before reaching the insert,
		// so the cleanup would be lost rather than repeated. Reconciliation
		// re-derives it from the profile's older succeeded export instead.
		//
		// [Ja] ここで失敗を報告すると、やることが残っていない試行を再試行することに
		// なる。リトライは終端状態の行を読んで投入に到達せず戻るため、掃除は繰り返され
		// るのではなく失われる。代わりにリコンシリエーションが、プロフィールの古い
		// succeeded から再導出する。
		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusSucceeded)
		}
		if _, ok := fixture.storage.object(fixture.objectKey); !ok {
			t.Errorf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
	})

	t.Run("1 ページに収まらない月も cursor を辿って全件を 1 回ずつ書き出す", func(t *testing.T) {
		t.Parallel()

		// One post past the page size is what makes the month's write loop carry
		// its cursor into a second read. Below that boundary the first read
		// already reaches the end, so a loop that dropped or repeated a page
		// would still produce the right archive.
		//
		// [Ja] ページサイズを 1 件超えることで、月の書き出しループが cursor を
		// 2 回目の取得へ引き継ぐ。この境界より下では初回の取得で終端に達するため、
		// ページを取りこぼしたり読み直したりするループでも正しいアーカイブが
		// できてしまう。
		postCount := int(usecase.ExportPostPageSize) + 1
		publishedAts := make([]time.Time, 0, postCount)
		for i := range postCount {
			// Spacing the posts a minute apart keeps the order their bodies
			// encode unambiguous and the whole set inside one month.
			//
			// [Ja] 投稿を 1 分ずつずらすことで、本文が表す順序が一意に決まり、全件が
			// 同じ月に収まる。
			publishedAts = append(publishedAts, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i)*time.Minute))
		}

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo", publishedAts...)

		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		data, ok := fixture.storage.object(fixture.objectKey)
		if !ok {
			t.Fatalf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
		got := postContents(zipEntryContent(t, data, "posts/2026-07.html"))
		if len(got) != postCount {
			t.Fatalf("書き出した投稿件数 = %d, want %d", len(got), postCount)
		}
		// Comparing position by position catches a page read twice or skipped,
		// which the count alone misses whenever the two cancel out.
		//
		// [Ja] 位置ごとに比較することで、ページの二重読みや取りこぼしを検出する。
		// 件数だけでは、両者が相殺したときに見逃す。
		for i, content := range got {
			if want := fmt.Sprintf("投稿 %d の本文", i); content != want {
				t.Fatalf("%d 件目の本文 = %q, want %q", i, content, want)
			}
		}
	})

	t.Run("試行を引き受けられなければ何もせず完了する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)

		// Take the row with another transition in the window between the read
		// that selected it and the update that would take it. Reconciliation
		// does this to an attempt it considers stale, and the attempt that lost
		// the row must not go on to build an archive for it.
		//
		// [Ja] 行を選んだ読み取りと、その行を引き受ける更新との間の窓で、別の遷移が
		// 行を奪う。リコンシリエーションが stale と判断した試行に対して行うことで、
		// 行を失った試行がその行のアーカイブを作り続けてはならない。
		faultExportRepo := repository.NewExportRepository(query.New(&faultInjectingDBTX{
			DBTX:              tx,
			beforeQueryRowFor: "-- name: MarkExportStarted",
			beforeQueryRow: func() {
				current := fixture.reload(t)
				if _, err := fixture.exportRepo.MarkStarted(ctx, current.ID, current.UpdatedAt); err != nil {
					t.Errorf("競合させるための MarkStarted() error = %v", err)
				}
			},
		}))
		uc := usecase.NewGenerateExportUsecase(
			faultExportRepo,
			fixture.postRepo,
			fixture.actorRepo,
			fixture.userRepo,
			allowingExportProfileDeletionGuard{},
			exportfile.NewBuilder(),
			fixture.storage,
			dispatcher.NewDispatcher(fixture.inserter),
		)

		// The job carries no work the export still needs, so reporting a failure
		// would only retry it against a row that belongs to another attempt.
		//
		// [Ja] このジョブにエクスポートが必要とする作業は残っていないため、失敗として
		// 報告しても、別の試行が保持する行に対してリトライを重ねるだけになる。
		if err := uc.Execute(ctx, usecase.GenerateExportInput{
			ExportID:       fixture.export.ID,
			IsFinalAttempt: true,
		}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d, want 1 (行を奪った遷移の分だけ)", got.AttemptCount)
		}
		if count := fixture.snapshotCount(t); count != 1 {
			t.Errorf("snapshot 件数 = %d, want 1", count)
		}
		if uploads := fixture.storage.uploads(); len(uploads) != 0 {
			t.Errorf("アップロードしたキー = %v, want none", uploads)
		}
		if deletes := fixture.storage.deletes(); len(deletes) != 0 {
			t.Errorf("削除したキー = %v, want none", deletes)
		}
	})

	t.Run("開始記録の DB エラーは試行として数えず最終試行でも収束させない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)
		faultExportRepo := repository.NewExportRepository(query.New(&faultInjectingDBTX{
			DBTX:             tx,
			queryRowErrorFor: "-- name: MarkExportStarted",
		}))
		uc := usecase.NewGenerateExportUsecase(
			faultExportRepo,
			fixture.postRepo,
			fixture.actorRepo,
			fixture.userRepo,
			allowingExportProfileDeletionGuard{},
			exportfile.NewBuilder(),
			fixture.storage,
			dispatcher.NewDispatcher(fixture.inserter),
		)

		err := uc.Execute(ctx, usecase.GenerateExportInput{
			ExportID:       fixture.export.ID,
			IsFinalAttempt: true,
		})
		if err == nil {
			t.Fatal("Execute() = nil, want an error")
		}

		// The transition that failed is the one that accepts the attempt, so
		// this attempt never held the export: it is neither counted nor closed,
		// and the export stays available to the next one.
		//
		// [Ja] 失敗したのは試行を引き受ける遷移そのものであるため、この試行は
		// エクスポートを保持していない。計上も終端化もせず、エクスポートは次の試行が
		// 引き受けられるまま残る。
		got := fixture.reload(t)
		if got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusQueued)
		}
		if got.AttemptCount != 0 {
			t.Errorf("got.AttemptCount = %d, want 0", got.AttemptCount)
		}
		if count := fixture.snapshotCount(t); count != 1 {
			t.Errorf("snapshot 件数 = %d, want 1", count)
		}
		if uploads := fixture.storage.uploads(); len(uploads) != 0 {
			t.Errorf("アップロードしたキー = %v, want none", uploads)
		}
		if deletes := fixture.storage.deletes(); len(deletes) != 0 {
			t.Errorf("削除したキー = %v, want none", deletes)
		}
	})

	t.Run("申請者の解決失敗も試行として数え最終試行なら failed へ収束する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)
		faultActorRepo := repository.NewActorRepository(query.New(&faultInjectingDBTX{
			DBTX:             tx,
			queryRowErrorFor: "-- name: GetActorByID",
		}))
		uc := usecase.NewGenerateExportUsecase(
			fixture.exportRepo,
			fixture.postRepo,
			faultActorRepo,
			fixture.userRepo,
			allowingExportProfileDeletionGuard{},
			exportfile.NewBuilder(),
			fixture.storage,
			dispatcher.NewDispatcher(fixture.inserter),
		)

		err := uc.Execute(ctx, usecase.GenerateExportInput{
			ExportID:       fixture.export.ID,
			IsFinalAttempt: true,
		})
		if err == nil {
			t.Fatal("Execute() = nil, want an error")
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusFailed)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d, want 1", got.AttemptCount)
		}
		if count := fixture.snapshotCount(t); count != 0 {
			t.Errorf("failed 後の snapshot 件数 = %d, want 0", count)
		}
		if uploads := fixture.storage.uploads(); len(uploads) != 0 {
			t.Errorf("アップロードしたキー = %v, want none", uploads)
		}
		if deletes := fixture.storage.deletes(); !slices.Equal(deletes, []string{fixture.objectKey}) {
			t.Errorf("削除したキー = %v, want %v", deletes, []string{fixture.objectKey})
		}
	})

	t.Run("アーカイブ書き出しの panic を試行失敗として収束させる", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			isFinalAttempt bool
			wantStatus     model.ExportStatus
			wantSnapshot   int64
			wantDelete     bool
		}{
			{
				name:           "最終試行前は started と snapshot を次の試行に残す",
				isFinalAttempt: false,
				wantStatus:     model.ExportStatusStarted,
				wantSnapshot:   1,
				wantDelete:     false,
			},
			{
				name:           "最終試行は failed へ収束して snapshot と object を削除する",
				isFinalAttempt: true,
				wantStatus:     model.ExportStatusFailed,
				wantSnapshot:   0,
				wantDelete:     true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, tx := testutil.SetupTx(t)
				fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
					time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
				)
				uc := usecase.NewGenerateExportUsecase(
					fixture.exportRepo,
					fixture.postRepo,
					fixture.actorRepo,
					fixture.userRepo,
					allowingExportProfileDeletionGuard{},
					panickingExportArchiveBuilder{},
					fixture.storage,
					dispatcher.NewDispatcher(fixture.inserter),
				)

				err := uc.Execute(ctx, usecase.GenerateExportInput{
					ExportID:       fixture.export.ID,
					IsFinalAttempt: tt.isFinalAttempt,
				})
				if err == nil {
					t.Fatal("Execute() = nil, want an error")
				}
				if !strings.Contains(err.Error(), "注入したアーカイブ panic") {
					t.Errorf("Execute() error = %v, want injected archive panic", err)
				}

				got := fixture.reload(t)
				if got.Status != tt.wantStatus {
					t.Errorf("got.Status = %v, want %v", got.Status, tt.wantStatus)
				}
				if got.AttemptCount != 1 {
					t.Errorf("got.AttemptCount = %d, want 1", got.AttemptCount)
				}
				if count := fixture.snapshotCount(t); count != tt.wantSnapshot {
					t.Errorf("snapshot 件数 = %d, want %d", count, tt.wantSnapshot)
				}
				if uploads := fixture.storage.uploads(); !slices.Equal(uploads, []string{fixture.objectKey}) {
					t.Errorf("アップロードしたキー = %v, want %v", uploads, []string{fixture.objectKey})
				}
				if _, ok := fixture.storage.object(fixture.objectKey); ok {
					t.Error("書き出しが panic したオブジェクトを保存した")
				}
				var wantDeletedKeys []string
				if tt.wantDelete {
					wantDeletedKeys = []string{fixture.objectKey}
				}
				if deletes := fixture.storage.deletes(); !slices.Equal(deletes, wantDeletedKeys) {
					t.Errorf("削除したキー = %v, want %v", deletes, wantDeletedKeys)
				}
			})
		}
	})

	t.Run("アップロードの panic でもアーカイブの書き出しを解放する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)
		builder := &blockingExportArchiveBuilder{writerExited: make(chan struct{})}
		fixture.storage.uploadHook = func(context.Context, io.Reader) error {
			panic("注入したアップロード panic")
		}
		uc := usecase.NewGenerateExportUsecase(
			fixture.exportRepo,
			fixture.postRepo,
			fixture.actorRepo,
			fixture.userRepo,
			allowingExportProfileDeletionGuard{},
			builder,
			fixture.storage,
			dispatcher.NewDispatcher(fixture.inserter),
		)

		// Upload runs without reading the pipe and panics while WriteIndex is
		// blocked. Execute intentionally leaves that panic to River, but its
		// deferred reader close must first release the archive goroutine.
		//
		// [Ja] Upload は pipe を読まず、WriteIndex がブロックしている間に panic する。
		// Execute は意図どおり panic を River に委ねるが、defer した読み取り側の close で
		// 先にアーカイブの goroutine を解放しなければならない。
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID})
		}()
		if got := fmt.Sprint(recovered); got != "注入したアップロード panic" {
			t.Fatalf("recover() = %q, want injected upload panic", got)
		}

		select {
		case <-builder.writerExited:
		case <-time.After(time.Second):
			t.Fatal("アップロードの panic 後もアーカイブの goroutine が終了しなかった")
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d, want 1", got.AttemptCount)
		}
		if count := fixture.snapshotCount(t); count != 1 {
			t.Errorf("snapshot 件数 = %d, want 1", count)
		}
		if uploads := fixture.storage.uploads(); !slices.Equal(uploads, []string{fixture.objectKey}) {
			t.Errorf("アップロードしたキー = %v, want %v", uploads, []string{fixture.objectKey})
		}
		if _, ok := fixture.storage.object(fixture.objectKey); ok {
			t.Error("panic したアップロードが partial object を保存した")
		}
	})

	t.Run("アップロード後の成功記録DBエラーを再試行可否に応じて収束させる", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			isFinalAttempt bool
			wantStatus     model.ExportStatus
			wantSnapshot   int64
			wantObject     bool
			wantDelete     bool
		}{
			{
				name:           "最終試行前は started とアーカイブを次の試行に残す",
				isFinalAttempt: false,
				wantStatus:     model.ExportStatusStarted,
				wantSnapshot:   1,
				wantObject:     true,
				wantDelete:     false,
			},
			{
				name:           "最終試行は failed へ収束してアーカイブを削除する",
				isFinalAttempt: true,
				wantStatus:     model.ExportStatusFailed,
				wantSnapshot:   0,
				wantObject:     false,
				wantDelete:     true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, tx := testutil.SetupTx(t)
				fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
					time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
				)
				faultExportRepo := repository.NewExportRepository(query.New(&faultInjectingDBTX{
					DBTX:         tx,
					execErrorFor: "-- name: MarkExportSucceeded",
				}))
				uc := usecase.NewGenerateExportUsecase(
					faultExportRepo,
					fixture.postRepo,
					fixture.actorRepo,
					fixture.userRepo,
					allowingExportProfileDeletionGuard{},
					exportfile.NewBuilder(),
					fixture.storage,
					dispatcher.NewDispatcher(fixture.inserter),
				)

				err := uc.Execute(ctx, usecase.GenerateExportInput{
					ExportID:       fixture.export.ID,
					IsFinalAttempt: tt.isFinalAttempt,
				})
				if err == nil {
					t.Fatal("Execute() = nil, want an error")
				}
				if !strings.Contains(err.Error(), "注入したデータベースエラー") {
					t.Errorf("Execute() error = %v, want injected database error", err)
				}

				got := fixture.reload(t)
				if got.Status != tt.wantStatus {
					t.Errorf("got.Status = %v, want %v", got.Status, tt.wantStatus)
				}
				if got.AttemptCount != 1 {
					t.Errorf("got.AttemptCount = %d, want 1", got.AttemptCount)
				}
				if count := fixture.snapshotCount(t); count != tt.wantSnapshot {
					t.Errorf("snapshot 件数 = %d, want %d", count, tt.wantSnapshot)
				}
				if _, ok := fixture.storage.object(fixture.objectKey); ok != tt.wantObject {
					t.Errorf("オブジェクトの存在 = %v, want %v", ok, tt.wantObject)
				}
				var wantDeletedKeys []string
				if tt.wantDelete {
					wantDeletedKeys = []string{fixture.objectKey}
				}
				if deletes := fixture.storage.deletes(); !slices.Equal(deletes, wantDeletedKeys) {
					t.Errorf("削除したキー = %v, want %v", deletes, wantDeletedKeys)
				}
			})
		}
	})

	t.Run("月の分割はユーザーのタイムゾーンで決まる", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			timeZone  string
			wantEntry string
		}{
			{
				name:      "解決できるタイムゾーンはその壁時計で月を決める",
				timeZone:  "Asia/Tokyo",
				wantEntry: "posts/2026-07.html",
			},
			// A zone PostgreSQL cannot resolve would fail the query in the
			// middle of a long generation, so an unresolvable name falls back
			// to UTC and the archive is still produced.
			//
			// [Ja] PostgreSQL が解決できないゾーンは長い生成の途中でクエリを失敗
			// させるため、解決できない名前は UTC へフォールバックし、アーカイブは
			// それでも生成される。
			{
				name:      "解決できないタイムゾーンは UTC にフォールバックする",
				timeZone:  "Invalid/Zone",
				wantEntry: "posts/2026-06.html",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, tx := testutil.SetupTx(t)
				// 2026-06-30T15:00:00Z is July 1st in Asia/Tokyo and still June
				// in UTC, so the entry name shows which zone was used.
				//
				// [Ja] 2026-06-30T15:00:00Z は Asia/Tokyo では 7 月 1 日、UTC では
				// まだ 6 月のため、エントリ名がどちらのゾーンを使ったかを示す。
				fixture := newGenerateExportFixture(t, tx, "ja", tt.timeZone,
					time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC),
				)

				if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
					t.Fatalf("Execute() error = %v", err)
				}

				data, ok := fixture.storage.object(fixture.objectKey)
				if !ok {
					t.Fatalf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
				}
				wantEntries := []string{"index.html", tt.wantEntry}
				if got := zipEntryNames(t, data); !slices.Equal(got, wantEntries) {
					t.Errorf("zip エントリ = %v, want %v", got, wantEntries)
				}
			})
		}
	})

	t.Run("アップロード後の状態更新が競合したら失敗し、再試行が同じキーへ上書きする", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)

		// Move the row on while the archive is being uploaded, so the attempt
		// finds its token stale when it tries to record the success. This is
		// the window between a stored object and the row that points at it.
		//
		// [Ja] アーカイブのアップロード中に行を進め、成功を記録しようとした試行が
		// 自分のトークンを古いものとして見つけるようにする。これがオブジェクトの
		// 保存と、それを指す行の更新との間の窓にあたる。
		fixture.storage.uploadHook = func(_ context.Context, body io.Reader) error {
			data, err := io.ReadAll(body)
			if err != nil {
				return err
			}
			fixture.storage.mu.Lock()
			fixture.storage.objects[fixture.objectKey] = data
			fixture.storage.mu.Unlock()

			current := fixture.reload(t)
			if _, err := fixture.exportRepo.MarkStarted(ctx, current.ID, current.UpdatedAt); err != nil {
				t.Errorf("競合させるための MarkStarted() error = %v", err)
			}
			return nil
		}

		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err == nil {
			t.Fatal("Execute() = nil, want an error")
		}
		if got := fixture.reload(t); got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusStarted)
		}
		if _, ok := fixture.storage.object(fixture.objectKey); !ok {
			t.Errorf("競合前にアップロードしたオブジェクトが残っていない (key: %s)", fixture.objectKey)
		}

		// The retry uses the same deterministic key, so it overwrites the
		// object the failed attempt left instead of adding a second one.
		//
		// [Ja] 再試行は同じ決定的なキーを使うため、失敗した試行が残したオブジェクトを
		// 2 つ目を増やすのではなく上書きする。
		fixture.storage.uploadHook = nil
		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("再試行の Execute() error = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("再試行後の got.Status = %v, want %v", got.Status, model.ExportStatusSucceeded)
		}
		wantUploads := []string{fixture.objectKey, fixture.objectKey}
		if uploads := fixture.storage.uploads(); !slices.Equal(uploads, wantUploads) {
			t.Errorf("アップロードしたキー = %v, want %v", uploads, wantUploads)
		}
	})

	t.Run("最終試行でなければ started のまま次の試行に残す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)
		fixture.storage.uploadHook = func(context.Context, io.Reader) error {
			return errors.New("アップロードの一時的な失敗")
		}

		err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID})
		if err == nil {
			t.Fatal("Execute() = nil, want an error")
		}
		// What stopped the attempt is the upload, so that is what the reported
		// error has to name. The error the pipe hands the archive goroutine
		// only unblocks it and would otherwise stand in front of the cause.
		//
		// [Ja] 試行を止めたのはアップロードであり、報告されるエラーはそれを示す必要が
		// ある。pipe がアーカイブの goroutine へ渡すエラーはそれを解放するだけのもので、
		// そのままでは原因の前に並んでしまう。
		if !strings.Contains(err.Error(), "アップロードの一時的な失敗") {
			t.Errorf("Execute() error = %v, want it to report the upload failure", err)
		}
		if strings.Contains(err.Error(), "アーカイブの書き出しを中断") {
			t.Errorf("Execute() error = %v, want it to leave out the pipe's stop error", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusStarted)
		}
		// The snapshot is the retry's input, so it must survive a failure that
		// is not terminal.
		//
		// [Ja] snapshot は再試行の入力であるため、終端でない失敗を越えて残る必要がある。
		if count := fixture.snapshotCount(t); count != 1 {
			t.Errorf("snapshot 件数 = %d, want 1", count)
		}
		if deletes := fixture.storage.deletes(); len(deletes) != 0 {
			t.Errorf("削除したキー = %v, want none", deletes)
		}
	})

	t.Run("最終試行の失敗は failed へ収束させオブジェクトを削除する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)
		// The upload stores a partial object and then fails, which is what
		// leaves an object behind for a terminal attempt to clean up.
		//
		// [Ja] アップロードが中途半端なオブジェクトを保存してから失敗する。これが、
		// 終端の試行が後始末すべきオブジェクトを残す状況にあたる。
		fixture.storage.uploadHook = func(_ context.Context, body io.Reader) error {
			data, err := io.ReadAll(body)
			if err != nil {
				return err
			}
			fixture.storage.mu.Lock()
			fixture.storage.objects[fixture.objectKey] = data
			fixture.storage.mu.Unlock()
			return errors.New("アップロードの恒久的な失敗")
		}

		err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{
			ExportID:       fixture.export.ID,
			IsFinalAttempt: true,
		})
		if err == nil {
			t.Fatal("Execute() = nil, want an error")
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusFailed)
		}
		if got.FinishedAt == nil {
			t.Error("got.FinishedAt should not be nil")
		}
		if count := fixture.snapshotCount(t); count != 0 {
			t.Errorf("failed 後の snapshot 件数 = %d, want 0", count)
		}
		if deletes := fixture.storage.deletes(); !slices.Equal(deletes, []string{fixture.objectKey}) {
			t.Errorf("削除したキー = %v, want %v", deletes, []string{fixture.objectKey})
		}
		if _, ok := fixture.storage.object(fixture.objectKey); ok {
			t.Errorf("失敗したエクスポートのオブジェクトが残っている (key: %s)", fixture.objectKey)
		}
	})

	t.Run("context がキャンセルされても最終試行は failed へ収束する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)

		attemptCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Cancel while the archive is still being written and stop reading it.
		// The archive goroutine is then blocked writing into the pipe, so a
		// generation that does not release it would hang this test.
		//
		// [Ja] アーカイブの書き出し中にキャンセルし、読み取りをやめる。アーカイブの
		// goroutine は pipe への書き込みでブロックするため、これを解放しない生成は
		// 本テストをハングさせる。
		fixture.storage.uploadHook = func(hookCtx context.Context, _ io.Reader) error {
			cancel()
			return hookCtx.Err()
		}

		err := fixture.uc.Execute(attemptCtx, usecase.GenerateExportInput{
			ExportID:       fixture.export.ID,
			IsFinalAttempt: true,
		})
		if err == nil {
			t.Fatal("Execute() = nil, want an error")
		}

		// The cleanup runs on a context detached from the canceled attempt, so
		// the export is closed instead of staying started forever.
		//
		// [Ja] 後処理はキャンセルされた試行から切り離した context で動くため、
		// エクスポートは started のまま残らずに閉じられる。
		got := fixture.reload(t)
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusFailed)
		}
		if deletes := fixture.storage.deletes(); !slices.Equal(deletes, []string{fixture.objectKey}) {
			t.Errorf("削除したキー = %v, want %v", deletes, []string{fixture.objectKey})
		}
	})

	t.Run("終端状態のエクスポートは再生成しない", func(t *testing.T) {
		t.Parallel()

		for _, status := range []model.ExportStatus{model.ExportStatusSucceeded, model.ExportStatusFailed} {
			t.Run(status.String(), func(t *testing.T) {
				t.Parallel()

				_, tx := testutil.SetupTx(t)
				fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
					time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
				)

				started, err := fixture.exportRepo.MarkStarted(ctx, fixture.export.ID, fixture.export.UpdatedAt)
				if err != nil || started == nil {
					t.Fatalf("MarkStarted() = (%v, %v), want (the started export, nil)", started, err)
				}
				if status == model.ExportStatusSucceeded {
					if _, err := fixture.exportRepo.MarkSucceeded(ctx, started.ID, fixture.objectKey, started.UpdatedAt); err != nil {
						t.Fatalf("MarkSucceeded() error = %v", err)
					}
				} else if _, err := fixture.exportRepo.MarkFailed(ctx, started.ID, started.UpdatedAt); err != nil {
					t.Fatalf("MarkFailed() error = %v", err)
				}
				before := fixture.reload(t)

				// A duplicate or replayed job must not restart a finished
				// export, or a success would be rebuilt as a failure.
				//
				// [Ja] 重複ジョブや再実行されたジョブが完了済みのエクスポートを
				// 再開してはならない。再開すると成功が失敗として作り直されうる。
				if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if uploads := fixture.storage.uploads(); len(uploads) != 0 {
					t.Errorf("アップロードしたキー = %v, want none", uploads)
				}

				got := fixture.reload(t)
				if got.Status != status {
					t.Errorf("got.Status = %v, want %v", got.Status, status)
				}
				if !got.UpdatedAt.Equal(before.UpdatedAt) {
					t.Errorf("got.UpdatedAt = %v, want %v (unchanged)", got.UpdatedAt, before.UpdatedAt)
				}
			})
		}
	})

	t.Run("存在しないエクスポートは何もせず完了する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo")

		// A job for a row that is gone has no outstanding work, so retrying it
		// until the attempts run out would only add noise.
		//
		// [Ja] 行が消えたジョブに未処理の作業は無いため、試行を使い切るまで
		// リトライしてもノイズが増えるだけになる。
		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: model.ExportID(testutil.MustParseUUID("00000000-0000-4000-8000-000000000000"))}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if uploads := fixture.storage.uploads(); len(uploads) != 0 {
			t.Errorf("アップロードしたキー = %v, want none", uploads)
		}
	})

	t.Run("投稿が無くても目次だけのアーカイブを生成する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "en", "UTC")

		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if got := fixture.reload(t); got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusSucceeded)
		}
		data, ok := fixture.storage.object(fixture.objectKey)
		if !ok {
			t.Fatalf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
		if got := zipEntryNames(t, data); !slices.Equal(got, []string{"index.html"}) {
			t.Errorf("zip エントリ = %v, want %v", got, []string{"index.html"})
		}
	})
}
