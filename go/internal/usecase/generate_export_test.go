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

// fakeExportObjectStorageはオブジェクトストレージの代役で、
// exports/{profile_id}/{export_id}.zipの外のキーをすべて拒否する。cleanupや
// 孤児回収が導出できないキーへ書く生成は、回収されないオブジェクトを残すのではなく
// テストの失敗になる。
type fakeExportObjectStorage struct {
	t         *testing.T
	profileID model.ProfileID
	exportID  model.ExportID

	// uploadHookを設定すると既定のアップロードを置き換える。テストは
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

// assertKeyは、キーがエクスポートの規約に従い、かつテスト対象のエクスポートを
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

// generateExportFixtureは生成可能な状態のエクスポート1件と、UseCaseが
// 使うリポジトリおよびfakeストレージ。
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

// faultInjectingDBTXは、テストが失敗させたい1つのクエリだけを失敗させ、
// それ以外は実トランザクションへ委譲する。対象は文の部分一致で選び、sqlcが出力する
// `-- name:` ヘッダーが安定した指定先になる。1文だけを失敗させることで、部分的な
// 失敗でしか生じない状態 (アップロード済みのアーカイブに対して、行をsucceededへ
// 進められない、など) を作る。
type faultInjectingDBTX struct {
	query.DBTX
	execErrorFor     string
	queryRowErrorFor string

	// beforeQueryRowはbeforeQueryRowForで選んだクエリの実行直前に走る。
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

// QueryRowContextは呼び出し側がScanできない行に差し替え、repositoryからは
// 取得エラーとして表面化させる。エラーを返すのではなく行を差し替えるのは、sql.Rowが
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

// newGenerateExportFixtureは指定したロケールとタイムゾーンのユーザー、指定した
// 時点に公開された投稿、およびそれらを固定化したqueuedのエクスポートを作成する。
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

// reloadはエクスポートの現在の行を返す。
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

// snapshotCountはエクスポートの申請時snapshotに残っている投稿の件数を返す。
func (f *generateExportFixture) snapshotCount(t *testing.T) int64 {
	t.Helper()

	months, err := f.postRepo.ListMonthsByExportID(context.Background(), repository.ListExportPostMonthsByExportIDInput{
		ExportID: f.export.ID,
		Location: time.UTC,
	})
	if err != nil {
		t.Fatalf("snapshotの取得に失敗: %v", err)
	}
	var total int64
	for _, month := range months {
		total += month.PostCount
	}
	return total
}

// zipEntryNamesはアップロードされたアーカイブに含まれるエントリ名を返す。
func zipEntryNames(t *testing.T, data []byte) []string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zipの読み取りに失敗: %v", err)
	}
	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	return names
}

// zipEntryContentはアップロードされたアーカイブの1エントリの内容を返す。
func zipEntryContent(t *testing.T, data []byte, name string) string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zipの読み取りに失敗: %v", err)
	}
	file, err := reader.Open(name)
	if err != nil {
		t.Fatalf("zipエントリ%qの取得に失敗: %v", name, err)
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("zipエントリ%qの読み取りに失敗: %v", name, err)
	}
	return string(content)
}

// postContentPatternは月のエントリに含まれるポスト1件の本文にマッチする。
// アーカイブのHTML契約は、それを描画するパッケージのパーサーによるテストが
// 固定しているため、ここで必要なのはドキュメント順に本文を復元することだけである。
var postContentPattern = regexp.MustCompile(`<p class="e-content">([^<]*)</p>`)

// postContentsは月のエントリに含まれる各ポストの本文を、ドキュメント順で
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

	t.Run("アーカイブをアップロードしてsucceededへ遷移しsnapshotを破棄する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
		)

		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusSucceeded)
		}
		if got.ObjectKey == nil || *got.ObjectKey != fixture.objectKey {
			t.Errorf("got.ObjectKey = %v、期待値 = %v", got.ObjectKey, fixture.objectKey)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d、期待値 = 1", got.AttemptCount)
		}

		// snapshotは試行のためだけに存在したため、試行より長く残ってはならない。
		if count := fixture.snapshotCount(t); count != 0 {
			t.Errorf("succeeded後のsnapshot件数 = %d、期待値 = 0", count)
		}

		data, ok := fixture.storage.object(fixture.objectKey)
		if !ok {
			t.Fatalf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
		wantEntries := []string{"index.html", "posts/2026-06.html", "posts/2026-07.html"}
		if got := zipEntryNames(t, data); !slices.Equal(got, wantEntries) {
			t.Errorf("zipエントリ = %v、期待値 = %v", got, wantEntries)
		}
		if content := zipEntryContent(t, data, "posts/2026-07.html"); !bytes.Contains([]byte(content), []byte("投稿 1 の本文")) {
			t.Errorf("posts/2026-07.htmlに投稿本文が含まれていない: %s", content)
		}

		// この成功はプロフィールがそれまで持っていたものからダウンロードを
		// 引き継ぐため、それらのエクスポートが占有し続けるストレージの解放を、この
		// 投入を裏で支えるリコンシリエーションを待たずに直ちに要求する。
		assertJobFor(t, fixture.inserter, "cleanup_old_exports", fixture.export.ProfileID.String())

		// 申請者はメールを待っているため、この成功が作成した通知は数分おきの
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

		// 通知は成功と同じ文で作成される、それ自体がdurableな行であるため、
		// 失われた投入はoutboxから再導出される。失敗を報告すると、終端状態の行を
		// 読んで何も投入せずに戻る試行を再試行することになる。
		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusSucceeded)
		}
		if _, ok := fixture.storage.object(fixture.objectKey); !ok {
			t.Errorf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
		// メールのジョブを投入できなくても、旧エクスポートの解放は行われる必要が
		// ある。2つの副作用はそれぞれ独立に収束する。
		assertJobFor(t, fixture.inserter, "cleanup_old_exports", fixture.export.ProfileID.String())
	})

	t.Run("旧エクスポート削除ジョブの投入に失敗しても成功を取り消さない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)
		fixture.inserter.failKinds["cleanup_old_exports"] = true

		// ここで失敗を報告すると、やることが残っていない試行を再試行することに
		// なる。リトライは終端状態の行を読んで投入に到達せず戻るため、掃除は繰り返され
		// るのではなく失われる。代わりにリコンシリエーションが、プロフィールの古い
		// succeededから再導出する。
		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusSucceeded)
		}
		if _, ok := fixture.storage.object(fixture.objectKey); !ok {
			t.Errorf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
	})

	t.Run("1ページに収まらない月もcursorを辿って全件を1回ずつ書き出す", func(t *testing.T) {
		t.Parallel()

		// ページサイズを1件超えることで、月の書き出しループがcursorを
		// 2回目の取得へ引き継ぐ。この境界より下では初回の取得で終端に達するため、
		// ページを取りこぼしたり読み直したりするループでも正しいアーカイブが
		// できてしまう。
		postCount := int(usecase.ExportPostPageSize) + 1
		publishedAts := make([]time.Time, 0, postCount)
		for i := range postCount {
			// 投稿を1分ずつずらすことで、本文が表す順序が一意に決まり、全件が
			// 同じ月に収まる。
			publishedAts = append(publishedAts, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i)*time.Minute))
		}

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo", publishedAts...)

		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		data, ok := fixture.storage.object(fixture.objectKey)
		if !ok {
			t.Fatalf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
		got := postContents(zipEntryContent(t, data, "posts/2026-07.html"))
		if len(got) != postCount {
			t.Fatalf("書き出した投稿件数 = %d、期待値 = %d", len(got), postCount)
		}
		// 位置ごとに比較することで、ページの二重読みや取りこぼしを検出する。
		// 件数だけでは、両者が相殺したときに見逃す。
		for i, content := range got {
			if want := fmt.Sprintf("投稿 %d の本文", i); content != want {
				t.Fatalf("%d件目の本文 = %q、期待値 = %q", i, content, want)
			}
		}
	})

	t.Run("試行を引き受けられなければ何もせず完了する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)

		// 行を選んだ読み取りと、その行を引き受ける更新との間の窓で、別の遷移が
		// 行を奪う。リコンシリエーションがstaleと判断した試行に対して行うことで、
		// 行を失った試行がその行のアーカイブを作り続けてはならない。
		faultExportRepo := repository.NewExportRepository(query.New(&faultInjectingDBTX{
			DBTX:              tx,
			beforeQueryRowFor: "-- name: MarkExportStarted",
			beforeQueryRow: func() {
				current := fixture.reload(t)
				if _, err := fixture.exportRepo.MarkStarted(ctx, current.ID, current.UpdatedAt); err != nil {
					t.Errorf("競合させるためのMarkStarted()のエラー = %v", err)
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

		// このジョブにエクスポートが必要とする作業は残っていないため、失敗として
		// 報告しても、別の試行が保持する行に対してリトライを重ねるだけになる。
		if err := uc.Execute(ctx, usecase.GenerateExportInput{
			ExportID:       fixture.export.ID,
			IsFinalAttempt: true,
		}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d、期待値 = 1 (行を奪った遷移の分だけ)", got.AttemptCount)
		}
		if count := fixture.snapshotCount(t); count != 1 {
			t.Errorf("snapshot件数 = %d、期待値 = 1", count)
		}
		if uploads := fixture.storage.uploads(); len(uploads) != 0 {
			t.Errorf("アップロードしたキー = %v、期待値 = なし", uploads)
		}
		if deletes := fixture.storage.deletes(); len(deletes) != 0 {
			t.Errorf("削除したキー = %v、期待値 = なし", deletes)
		}
	})

	t.Run("開始記録のDBエラーは試行として数えず最終試行でも収束させない", func(t *testing.T) {
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
			t.Fatal("Execute()のエラー = nil、エラーを期待")
		}

		// 失敗したのは試行を引き受ける遷移そのものであるため、この試行は
		// エクスポートを保持していない。計上も終端化もせず、エクスポートは次の試行が
		// 引き受けられるまま残る。
		got := fixture.reload(t)
		if got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusQueued)
		}
		if got.AttemptCount != 0 {
			t.Errorf("got.AttemptCount = %d、期待値 = 0", got.AttemptCount)
		}
		if count := fixture.snapshotCount(t); count != 1 {
			t.Errorf("snapshot件数 = %d、期待値 = 1", count)
		}
		if uploads := fixture.storage.uploads(); len(uploads) != 0 {
			t.Errorf("アップロードしたキー = %v、期待値 = なし", uploads)
		}
		if deletes := fixture.storage.deletes(); len(deletes) != 0 {
			t.Errorf("削除したキー = %v、期待値 = なし", deletes)
		}
	})

	t.Run("申請者の解決失敗も試行として数え最終試行ならfailedへ収束する", func(t *testing.T) {
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
			t.Fatal("Execute()のエラー = nil、エラーを期待")
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusFailed)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d、期待値 = 1", got.AttemptCount)
		}
		if count := fixture.snapshotCount(t); count != 0 {
			t.Errorf("failed後のsnapshot件数 = %d、期待値 = 0", count)
		}
		if uploads := fixture.storage.uploads(); len(uploads) != 0 {
			t.Errorf("アップロードしたキー = %v、期待値 = なし", uploads)
		}
		if deletes := fixture.storage.deletes(); !slices.Equal(deletes, []string{fixture.objectKey}) {
			t.Errorf("削除したキー = %v、期待値 = %v", deletes, []string{fixture.objectKey})
		}
	})

	t.Run("アーカイブ書き出しのpanicを試行失敗として収束させる", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			isFinalAttempt bool
			wantStatus     model.ExportStatus
			wantSnapshot   int64
			wantDelete     bool
		}{
			{
				name:           "最終試行前はstartedとsnapshotを次の試行に残す",
				isFinalAttempt: false,
				wantStatus:     model.ExportStatusStarted,
				wantSnapshot:   1,
				wantDelete:     false,
			},
			{
				name:           "最終試行はfailedへ収束してsnapshotとobjectを削除する",
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
					t.Fatal("Execute()のエラー = nil、エラーを期待")
				}
				if !strings.Contains(err.Error(), "注入したアーカイブ panic") {
					t.Errorf("Execute()のエラー = %v、注入したアーカイブのpanicを期待", err)
				}

				got := fixture.reload(t)
				if got.Status != tt.wantStatus {
					t.Errorf("got.Status = %v、期待値 = %v", got.Status, tt.wantStatus)
				}
				if got.AttemptCount != 1 {
					t.Errorf("got.AttemptCount = %d、期待値 = 1", got.AttemptCount)
				}
				if count := fixture.snapshotCount(t); count != tt.wantSnapshot {
					t.Errorf("snapshot件数 = %d、期待値 = %d", count, tt.wantSnapshot)
				}
				if uploads := fixture.storage.uploads(); !slices.Equal(uploads, []string{fixture.objectKey}) {
					t.Errorf("アップロードしたキー = %v、期待値 = %v", uploads, []string{fixture.objectKey})
				}
				if _, ok := fixture.storage.object(fixture.objectKey); ok {
					t.Error("書き出しがpanicしたオブジェクトを保存した")
				}
				var wantDeletedKeys []string
				if tt.wantDelete {
					wantDeletedKeys = []string{fixture.objectKey}
				}
				if deletes := fixture.storage.deletes(); !slices.Equal(deletes, wantDeletedKeys) {
					t.Errorf("削除したキー = %v、期待値 = %v", deletes, wantDeletedKeys)
				}
			})
		}
	})

	t.Run("アップロードのpanicでもアーカイブの書き出しを解放する", func(t *testing.T) {
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

		// Uploadはpipeを読まず、WriteIndexがブロックしている間にpanicする。
		// Executeは意図どおりpanicをRiverに委ねるが、deferした読み取り側のcloseで
		// 先にアーカイブのgoroutineを解放しなければならない。
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID})
		}()
		if got := fmt.Sprint(recovered); got != "注入したアップロード panic" {
			t.Fatalf("recover() = %q、注入したアップロードのpanicを期待", got)
		}

		select {
		case <-builder.writerExited:
		case <-time.After(time.Second):
			t.Fatal("アップロードのpanic後もアーカイブのgoroutineが終了しなかった")
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d、期待値 = 1", got.AttemptCount)
		}
		if count := fixture.snapshotCount(t); count != 1 {
			t.Errorf("snapshot件数 = %d、期待値 = 1", count)
		}
		if uploads := fixture.storage.uploads(); !slices.Equal(uploads, []string{fixture.objectKey}) {
			t.Errorf("アップロードしたキー = %v、期待値 = %v", uploads, []string{fixture.objectKey})
		}
		if _, ok := fixture.storage.object(fixture.objectKey); ok {
			t.Error("panicしたアップロードがpartial objectを保存した")
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
				name:           "最終試行前はstartedとアーカイブを次の試行に残す",
				isFinalAttempt: false,
				wantStatus:     model.ExportStatusStarted,
				wantSnapshot:   1,
				wantObject:     true,
				wantDelete:     false,
			},
			{
				name:           "最終試行はfailedへ収束してアーカイブを削除する",
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
					t.Fatal("Execute()のエラー = nil、エラーを期待")
				}
				if !strings.Contains(err.Error(), "注入したデータベースエラー") {
					t.Errorf("Execute()のエラー = %v、注入したデータベースのエラーを期待", err)
				}

				got := fixture.reload(t)
				if got.Status != tt.wantStatus {
					t.Errorf("got.Status = %v、期待値 = %v", got.Status, tt.wantStatus)
				}
				if got.AttemptCount != 1 {
					t.Errorf("got.AttemptCount = %d、期待値 = 1", got.AttemptCount)
				}
				if count := fixture.snapshotCount(t); count != tt.wantSnapshot {
					t.Errorf("snapshot件数 = %d、期待値 = %d", count, tt.wantSnapshot)
				}
				if _, ok := fixture.storage.object(fixture.objectKey); ok != tt.wantObject {
					t.Errorf("オブジェクトの存在 = %v、期待値 = %v", ok, tt.wantObject)
				}
				var wantDeletedKeys []string
				if tt.wantDelete {
					wantDeletedKeys = []string{fixture.objectKey}
				}
				if deletes := fixture.storage.deletes(); !slices.Equal(deletes, wantDeletedKeys) {
					t.Errorf("削除したキー = %v、期待値 = %v", deletes, wantDeletedKeys)
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
			// PostgreSQLが解決できないゾーンは長い生成の途中でクエリを失敗
			// させるため、解決できない名前はUTCへフォールバックし、アーカイブは
			// それでも生成される。
			{
				name:      "解決できないタイムゾーンはUTCにフォールバックする",
				timeZone:  "Invalid/Zone",
				wantEntry: "posts/2026-06.html",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, tx := testutil.SetupTx(t)
				// 2026-06-30T15:00:00ZはAsia/Tokyoでは7月1日、UTCでは
				// まだ6月のため、エントリ名がどちらのゾーンを使ったかを示す。
				fixture := newGenerateExportFixture(t, tx, "ja", tt.timeZone,
					time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC),
				)

				if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
					t.Fatalf("Execute()のエラー = %v", err)
				}

				data, ok := fixture.storage.object(fixture.objectKey)
				if !ok {
					t.Fatalf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
				}
				wantEntries := []string{"index.html", tt.wantEntry}
				if got := zipEntryNames(t, data); !slices.Equal(got, wantEntries) {
					t.Errorf("zipエントリ = %v、期待値 = %v", got, wantEntries)
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

		// アーカイブのアップロード中に行を進め、成功を記録しようとした試行が
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
				t.Errorf("競合させるためのMarkStarted()のエラー = %v", err)
			}
			return nil
		}

		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err == nil {
			t.Fatal("Execute()のエラー = nil、エラーを期待")
		}
		if got := fixture.reload(t); got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusStarted)
		}
		if _, ok := fixture.storage.object(fixture.objectKey); !ok {
			t.Errorf("競合前にアップロードしたオブジェクトが残っていない (key: %s)", fixture.objectKey)
		}

		// 再試行は同じ決定的なキーを使うため、失敗した試行が残したオブジェクトを
		// 2つ目を増やすのではなく上書きする。
		fixture.storage.uploadHook = nil
		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("再試行のExecute()のエラー = %v", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("再試行後のgot.Status = %v、期待値 = %v", got.Status, model.ExportStatusSucceeded)
		}
		wantUploads := []string{fixture.objectKey, fixture.objectKey}
		if uploads := fixture.storage.uploads(); !slices.Equal(uploads, wantUploads) {
			t.Errorf("アップロードしたキー = %v、期待値 = %v", uploads, wantUploads)
		}
	})

	t.Run("最終試行でなければstartedのまま次の試行に残す", func(t *testing.T) {
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
			t.Fatal("Execute()のエラー = nil、エラーを期待")
		}
		// 試行を止めたのはアップロードであり、報告されるエラーはそれを示す必要が
		// ある。pipeがアーカイブのgoroutineへ渡すエラーはそれを解放するだけのもので、
		// そのままでは原因の前に並んでしまう。
		if !strings.Contains(err.Error(), "アップロードの一時的な失敗") {
			t.Errorf("Execute()のエラー = %v、アップロードの失敗を報告することを期待", err)
		}
		if strings.Contains(err.Error(), "アーカイブの書き出しを中断") {
			t.Errorf("Execute()のエラー = %v、パイプの停止エラーを含めないことを期待", err)
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusStarted)
		}
		// snapshotは再試行の入力であるため、終端でない失敗を越えて残る必要がある。
		if count := fixture.snapshotCount(t); count != 1 {
			t.Errorf("snapshot件数 = %d、期待値 = 1", count)
		}
		if deletes := fixture.storage.deletes(); len(deletes) != 0 {
			t.Errorf("削除したキー = %v、期待値 = なし", deletes)
		}
	})

	t.Run("最終試行の失敗はfailedへ収束させオブジェクトを削除する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)
		// アップロードが中途半端なオブジェクトを保存してから失敗する。これが、
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
			t.Fatal("Execute()のエラー = nil、エラーを期待")
		}

		got := fixture.reload(t)
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusFailed)
		}
		if got.FinishedAt == nil {
			t.Error("got.FinishedAt = nil、非nilを期待")
		}
		if count := fixture.snapshotCount(t); count != 0 {
			t.Errorf("failed後のsnapshot件数 = %d、期待値 = 0", count)
		}
		if deletes := fixture.storage.deletes(); !slices.Equal(deletes, []string{fixture.objectKey}) {
			t.Errorf("削除したキー = %v、期待値 = %v", deletes, []string{fixture.objectKey})
		}
		if _, ok := fixture.storage.object(fixture.objectKey); ok {
			t.Errorf("失敗したエクスポートのオブジェクトが残っている (key: %s)", fixture.objectKey)
		}
	})

	t.Run("contextがキャンセルされても最終試行はfailedへ収束する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo",
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		)

		attemptCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// アーカイブの書き出し中にキャンセルし、読み取りをやめる。アーカイブの
		// goroutineはpipeへの書き込みでブロックするため、これを解放しない生成は
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
			t.Fatal("Execute()のエラー = nil、エラーを期待")
		}

		// 後処理はキャンセルされた試行から切り離したcontextで動くため、
		// エクスポートはstartedのまま残らずに閉じられる。
		got := fixture.reload(t)
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusFailed)
		}
		if deletes := fixture.storage.deletes(); !slices.Equal(deletes, []string{fixture.objectKey}) {
			t.Errorf("削除したキー = %v、期待値 = %v", deletes, []string{fixture.objectKey})
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
					t.Fatalf("MarkStarted() = (%v, %v)、期待値 = (startedのエクスポート, nil)", started, err)
				}
				if status == model.ExportStatusSucceeded {
					if _, err := fixture.exportRepo.MarkSucceeded(ctx, started.ID, fixture.objectKey, started.UpdatedAt); err != nil {
						t.Fatalf("MarkSucceeded()のエラー = %v", err)
					}
				} else if _, err := fixture.exportRepo.MarkFailed(ctx, started.ID, started.UpdatedAt); err != nil {
					t.Fatalf("MarkFailed()のエラー = %v", err)
				}
				before := fixture.reload(t)

				// 重複ジョブや再実行されたジョブが完了済みのエクスポートを
				// 再開してはならない。再開すると成功が失敗として作り直されうる。
				if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
					t.Fatalf("Execute()のエラー = %v", err)
				}
				if uploads := fixture.storage.uploads(); len(uploads) != 0 {
					t.Errorf("アップロードしたキー = %v、期待値 = なし", uploads)
				}

				got := fixture.reload(t)
				if got.Status != status {
					t.Errorf("got.Status = %v、期待値 = %v", got.Status, status)
				}
				if !got.UpdatedAt.Equal(before.UpdatedAt) {
					t.Errorf("got.UpdatedAt = %v、期待値 = %v (変化しない)", got.UpdatedAt, before.UpdatedAt)
				}
			})
		}
	})

	t.Run("存在しないエクスポートは何もせず完了する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "ja", "Asia/Tokyo")

		// 行が消えたジョブに未処理の作業は無いため、試行を使い切るまで
		// リトライしてもノイズが増えるだけになる。
		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: model.ExportID(testutil.MustParseUUID("00000000-0000-4000-8000-000000000000"))}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if uploads := fixture.storage.uploads(); len(uploads) != 0 {
			t.Errorf("アップロードしたキー = %v、期待値 = なし", uploads)
		}
	})

	t.Run("投稿が無くても目次だけのアーカイブを生成する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newGenerateExportFixture(t, tx, "en", "UTC")

		if err := fixture.uc.Execute(ctx, usecase.GenerateExportInput{ExportID: fixture.export.ID}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if got := fixture.reload(t); got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusSucceeded)
		}
		data, ok := fixture.storage.object(fixture.objectKey)
		if !ok {
			t.Fatalf("オブジェクトが保存されていない (key: %s)", fixture.objectKey)
		}
		if got := zipEntryNames(t, data); !slices.Equal(got, []string{"index.html"}) {
			t.Errorf("zipエントリ = %v、期待値 = %v", got, []string{"index.html"})
		}
	})
}
