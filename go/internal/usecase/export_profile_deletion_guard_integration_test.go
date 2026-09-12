package usecase_test

import (
	"context"
	"database/sql"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/exportfile"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

const exportProfileDeletionRaceTimeout = 10 * time.Second

type committedExportDeletionTarget struct {
	db        *sql.DB
	userID    model.UserID
	profileID model.ProfileID
	actorID   model.ActorID
}

func newCommittedExportDeletionTarget(t *testing.T) committedExportDeletionTarget {
	t.Helper()

	db := testutil.GetTestDB()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("前提データ用 transaction の開始に失敗: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	userID := testutil.NewUserBuilder(t, tx).Build()
	profileID := testutil.NewProfileBuilder(t, tx).Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()
	if err := tx.Commit(); err != nil {
		t.Fatalf("前提データの commit に失敗: %v", err)
	}

	target := committedExportDeletionTarget{
		db:        db,
		userID:    userID,
		profileID: profileID,
		actorID:   actorID,
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM export_completion_notifications WHERE actor_id = $1", uuid.UUID(actorID))
		_, _ = db.Exec("DELETE FROM export_posts WHERE export_id IN (SELECT id FROM exports WHERE profile_id = $1)", uuid.UUID(profileID))
		_, _ = db.Exec("DELETE FROM exports WHERE profile_id = $1", uuid.UUID(profileID))
		_, _ = db.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(actorID))
		_, _ = db.Exec("DELETE FROM profiles WHERE id = $1", uuid.UUID(profileID))
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", uuid.UUID(userID))
	})
	return target
}

func receiveExportProfileDeletionRaceResult(t *testing.T, label string, results <-chan error) error {
	t.Helper()

	select {
	case err := <-results:
		return err
	case <-time.After(exportProfileDeletionRaceTimeout):
		t.Fatalf("%s が完了しない", label)
		return nil
	}
}

func waitForExportProfileDeletionMarker(t *testing.T, target committedExportDeletionTarget) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), exportProfileDeletionRaceTimeout)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		var started bool
		err := target.db.QueryRowContext(ctx,
			"SELECT export_deletion_started_at IS NOT NULL FROM profiles WHERE id = $1",
			uuid.UUID(target.profileID),
		).Scan(&started)
		if err != nil {
			t.Fatalf("プロフィール削除マーカーの取得に失敗: %v", err)
		}
		if started {
			return
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("プロフィール削除マーカーが記録されない: %v", ctx.Err())
		}
	}
}

func assertCreateRejectedAfterExportDeletionStarted(t *testing.T, target committedExportDeletionTarget) {
	t.Helper()

	repo := repository.NewExportRepository(query.New(target.db))
	export, err := repo.Create(context.Background(), repository.CreateExportInput{
		ProfileID: target.profileID,
		ActorID:   target.actorID,
	})
	if err != nil {
		t.Fatalf("削除開始後の Create() error = %v, want nil", err)
	}
	if export != nil {
		t.Errorf("削除開始後の Create() export = %v, want nil", export)
	}
}

func TestExportProfileDeletionGuard_CreateAndDeleteSerialize(t *testing.T) {
	t.Parallel()

	target := newCommittedExportDeletionTarget(t)
	ctx, cancel := context.WithTimeout(context.Background(), exportProfileDeletionRaceTimeout)
	defer cancel()

	createTx, err := target.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Create 用 transaction の開始に失敗: %v", err)
	}
	defer func() { _ = createTx.Rollback() }()

	createRepo := repository.NewExportRepository(query.New(createTx))
	export, err := createRepo.Create(ctx, repository.CreateExportInput{
		ProfileID: target.profileID,
		ActorID:   target.actorID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	queries := query.New(target.db)
	exportRepo := repository.NewExportRepository(queries)
	notificationRepo := repository.NewExportCompletionNotificationRepository(queries)
	guardRepo := repository.NewExportProfileDeletionGuardRepository(target.db)
	storage := newFakeExportObjectStorage(t, target.profileID, export.ID)
	deleteUC := usecase.NewDeleteExportsForProfileUsecase(exportRepo, notificationRepo, guardRepo, storage)

	deleteStarted := make(chan struct{})
	deleteResults := make(chan error, 1)
	go func() {
		close(deleteStarted)
		deleteResults <- deleteUC.Execute(ctx, target.profileID)
	}()
	<-deleteStarted

	// CreateExport's profile-row lock keeps deletion from establishing its
	// boundary until the export and its snapshot commit together.
	//
	// [Ja] CreateExport のプロフィール行 lock により、export と snapshot がまとめて
	// commit されるまで、削除はその境界を確立できない。
	select {
	case err := <-deleteResults:
		t.Fatalf("Create の commit 前に Delete が完了した: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	if err := createTx.Commit(); err != nil {
		t.Fatalf("Create の commit に失敗: %v", err)
	}
	if err := receiveExportProfileDeletionRaceResult(t, "Delete", deleteResults); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if got, err := exportRepo.FindByID(ctx, export.ID); err != nil || got != nil {
		t.Errorf("Delete 後の FindByID() = (%v, %v), want (nil, nil)", got, err)
	}
	if _, ok := storage.object(usecase.ExportObjectKey(target.profileID, export.ID)); ok {
		t.Error("Delete 後もオブジェクトが残っている")
	}
	assertCreateRejectedAfterExportDeletionStarted(t, target)
}

func TestExportProfileDeletionGuard_GenerateAndDeleteSerialize(t *testing.T) {
	t.Parallel()

	target := newCommittedExportDeletionTarget(t)
	ctx, cancel := context.WithTimeout(context.Background(), exportProfileDeletionRaceTimeout)
	defer cancel()

	queries := query.New(target.db)
	exportRepo := repository.NewExportRepository(queries)
	export, err := exportRepo.Create(ctx, repository.CreateExportInput{
		ProfileID: target.profileID,
		ActorID:   target.actorID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	storage := newFakeExportObjectStorage(t, target.profileID, export.ID)
	uploadReady := make(chan struct{})
	finishUpload := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-finishUpload:
		default:
			close(finishUpload)
		}
	})
	storage.uploadHook = func(ctx context.Context, body io.Reader) error {
		data, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		close(uploadReady)
		select {
		case <-finishUpload:
		case <-ctx.Done():
			return ctx.Err()
		}
		storage.mu.Lock()
		storage.objects[usecase.ExportObjectKey(target.profileID, export.ID)] = data
		storage.mu.Unlock()
		return nil
	}

	guardRepo := repository.NewExportProfileDeletionGuardRepository(target.db)
	generateUC := usecase.NewGenerateExportUsecase(
		exportRepo,
		repository.NewExportPostRepository(queries),
		repository.NewActorRepository(queries),
		repository.NewUserRepository(queries),
		guardRepo,
		exportfile.NewBuilder(),
		storage,
		dispatcher.NewDispatcher(newExportJobInserter(t)),
	)
	notificationRepo := repository.NewExportCompletionNotificationRepository(queries)
	deleteUC := usecase.NewDeleteExportsForProfileUsecase(exportRepo, notificationRepo, guardRepo, storage)

	generateResults := make(chan error, 1)
	go func() {
		generateResults <- generateUC.Execute(ctx, usecase.GenerateExportInput{ExportID: export.ID})
	}()
	select {
	case <-uploadReady:
	case <-ctx.Done():
		t.Fatalf("upload が完了直前まで進まない: %v", ctx.Err())
	}

	deleteResults := make(chan error, 1)
	go func() {
		deleteResults <- deleteUC.Execute(ctx, target.profileID)
	}()
	waitForExportProfileDeletionMarker(t, target)
	if err := generateUC.Execute(ctx, usecase.GenerateExportInput{ExportID: export.ID}); err != nil {
		t.Fatalf("削除開始後の Generate() error = %v", err)
	}
	if got := storage.uploads(); len(got) != 1 {
		t.Errorf("削除開始後に upload が増えた: %v", got)
	}

	select {
	case err := <-deleteResults:
		t.Fatalf("upload の再開前に Delete が完了した: %v", err)
	default:
	}
	close(finishUpload)

	if err := receiveExportProfileDeletionRaceResult(t, "Generate", generateResults); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := receiveExportProfileDeletionRaceResult(t, "Delete", deleteResults); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	objectKey := usecase.ExportObjectKey(target.profileID, export.ID)
	if _, ok := storage.object(objectKey); ok {
		t.Error("Delete の成功後も upload 済みオブジェクトが残っている")
	}
	if got, err := exportRepo.FindByID(ctx, export.ID); err != nil || got != nil {
		t.Errorf("Delete 後の FindByID() = (%v, %v), want (nil, nil)", got, err)
	}
	if got, err := notificationRepo.FindByExportID(ctx, export.ID); err != nil || got != nil {
		t.Errorf("Delete 後の FindByExportID() = (%v, %v), want (nil, nil)", got, err)
	}
	assertCreateRejectedAfterExportDeletionStarted(t, target)
}

// newCommittedSucceededExport creates the database and outbox state that
// completion delivery and profile deletion race over.
//
// [Ja] newCommittedSucceededExport は完了通知の配信とプロフィール削除が競合する
// DB・outbox 状態を作成する。
func newCommittedSucceededExport(
	t *testing.T,
	target committedExportDeletionTarget,
) (*repository.ExportRepository, *repository.ExportCompletionNotificationRepository, model.ExportID, string) {
	t.Helper()

	queries := query.New(target.db)
	exportRepo := repository.NewExportRepository(queries)
	notificationRepo := repository.NewExportCompletionNotificationRepository(queries)
	export, err := exportRepo.Create(context.Background(), repository.CreateExportInput{
		ProfileID: target.profileID,
		ActorID:   target.actorID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	started, err := exportRepo.MarkStarted(context.Background(), export.ID, export.UpdatedAt)
	if err != nil {
		t.Fatalf("MarkStarted() error = %v", err)
	}
	if started == nil {
		t.Fatal("MarkStarted() = nil, want started export")
	}

	objectKey := usecase.ExportObjectKey(target.profileID, export.ID)
	succeeded, err := exportRepo.MarkSucceeded(context.Background(), export.ID, objectKey, started.UpdatedAt)
	if err != nil {
		t.Fatalf("MarkSucceeded() error = %v", err)
	}
	if !succeeded {
		t.Fatal("MarkSucceeded() = false, want true")
	}
	return exportRepo, notificationRepo, export.ID, objectKey
}

type blockingExportCompletedSender struct {
	started     chan struct{}
	finish      chan struct{}
	startedOnce sync.Once
	finishOnce  sync.Once
	mu          sync.Mutex
	sendCount   int
}

func newBlockingExportCompletedSender() *blockingExportCompletedSender {
	return &blockingExportCompletedSender{
		started: make(chan struct{}),
		finish:  make(chan struct{}),
	}
}

func (s *blockingExportCompletedSender) Send(ctx context.Context, _, _, _, _ string) error {
	s.mu.Lock()
	s.sendCount++
	s.mu.Unlock()
	s.startedOnce.Do(func() { close(s.started) })

	select {
	case <-s.finish:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *blockingExportCompletedSender) unblock() {
	s.finishOnce.Do(func() { close(s.finish) })
}

func (s *blockingExportCompletedSender) sends() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sendCount
}

// pausingExportProfileDeletionGuard stops a notification delivery after it read
// the notification but before it enters the real shared guard.
//
// [Ja] pausingExportProfileDeletionGuard は通知の読み取り後、実際の共有 guard に
// 入る前で通知配信を停止する。
type pausingExportProfileDeletionGuard struct {
	delegate    usecase.ExportProfileDeletionGuard
	reached     chan struct{}
	proceed     chan struct{}
	reachedOnce sync.Once
	proceedOnce sync.Once
}

func newPausingExportProfileDeletionGuard(
	delegate usecase.ExportProfileDeletionGuard,
) *pausingExportProfileDeletionGuard {
	return &pausingExportProfileDeletionGuard{
		delegate: delegate,
		reached:  make(chan struct{}),
		proceed:  make(chan struct{}),
	}
}

func (g *pausingExportProfileDeletionGuard) BeginOperation(
	ctx context.Context,
	profileID model.ProfileID,
) (func() error, bool, error) {
	g.reachedOnce.Do(func() { close(g.reached) })
	select {
	case <-g.proceed:
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}
	return g.delegate.BeginOperation(ctx, profileID)
}

func (g *pausingExportProfileDeletionGuard) BeginDeletion(
	ctx context.Context,
	profileID model.ProfileID,
) (func() error, bool, error) {
	return g.delegate.BeginDeletion(ctx, profileID)
}

func (g *pausingExportProfileDeletionGuard) unblock() {
	g.proceedOnce.Do(func() { close(g.proceed) })
}

// TestExportProfileDeletionGuard_SendCompletionEmailAndDeleteSerialize proves
// that profile deletion cannot remove the archive, export, or outbox while a
// completion delivery owns the profile's shared export-operation lock.
//
// [Ja] TestExportProfileDeletionGuard_SendCompletionEmailAndDeleteSerialize は、
// 完了通知の配信がプロフィールの export 操作用共有 lock を保持している間、
// プロフィール削除がアーカイブ・export・outbox を削除できないことを検証する。
func TestExportProfileDeletionGuard_SendCompletionEmailAndDeleteSerialize(t *testing.T) {
	t.Parallel()

	target := newCommittedExportDeletionTarget(t)
	ctx, cancel := context.WithTimeout(context.Background(), exportProfileDeletionRaceTimeout)
	defer cancel()

	exportRepo, notificationRepo, exportID, objectKey := newCommittedSucceededExport(t, target)
	guardRepo := repository.NewExportProfileDeletionGuardRepository(target.db)
	storage := newFakeExportObjectStorage(t, target.profileID, exportID)
	storage.objects[objectKey] = []byte("archive")
	sender := newBlockingExportCompletedSender()
	t.Cleanup(sender.unblock)

	sendUC := usecase.NewSendExportCompletedEmailUsecase(
		notificationRepo,
		guardRepo,
		sender,
		exportPageURL,
	)
	deleteUC := usecase.NewDeleteExportsForProfileUsecase(exportRepo, notificationRepo, guardRepo, storage)

	sendResults := make(chan error, 1)
	go func() {
		sendResults <- sendUC.Execute(ctx, exportID)
	}()
	select {
	case <-sender.started:
	case <-ctx.Done():
		t.Fatalf("sender が呼ばれない: %v", ctx.Err())
	}

	deleteResults := make(chan error, 1)
	go func() {
		deleteResults <- deleteUC.Execute(ctx, target.profileID)
	}()
	waitForExportProfileDeletionMarker(t, target)

	select {
	case err := <-deleteResults:
		t.Fatalf("sender の再開前に Delete が完了した: %v", err)
	default:
	}
	if got, err := exportRepo.FindByID(ctx, exportID); err != nil || got == nil {
		t.Errorf("sender 停止中の FindByID() = (%v, %v), want (export, nil)", got, err)
	}
	if got, err := notificationRepo.FindByExportID(ctx, exportID); err != nil || got == nil {
		t.Errorf("sender 停止中の FindByExportID() = (%v, %v), want (notification, nil)", got, err)
	}
	if _, ok := storage.object(objectKey); !ok {
		t.Error("sender 停止中にオブジェクトが削除された")
	}

	sender.unblock()
	if err := receiveExportProfileDeletionRaceResult(t, "Send", sendResults); err != nil {
		t.Fatalf("Send Execute() error = %v", err)
	}
	if err := receiveExportProfileDeletionRaceResult(t, "Delete", deleteResults); err != nil {
		t.Fatalf("Delete Execute() error = %v", err)
	}

	if got := sender.sends(); got != 1 {
		t.Errorf("送信件数 = %d, want 1", got)
	}
	if got, err := exportRepo.FindByID(ctx, exportID); err != nil || got != nil {
		t.Errorf("Delete 後の FindByID() = (%v, %v), want (nil, nil)", got, err)
	}
	if got, err := notificationRepo.FindByExportID(ctx, exportID); err != nil || got != nil {
		t.Errorf("Delete 後の FindByExportID() = (%v, %v), want (nil, nil)", got, err)
	}
	if _, ok := storage.object(objectKey); ok {
		t.Error("Delete 後もオブジェクトが残っている")
	}
}

// TestExportProfileDeletionGuard_DeleteWinsAfterNotificationSnapshot proves
// that a deletion boundary established after the delivery read its snapshot
// still prevents that stale snapshot from reaching the mail provider.
//
// [Ja] TestExportProfileDeletionGuard_DeleteWinsAfterNotificationSnapshot は、
// 配信が snapshot を読み取った後でも、削除境界が先に確立されれば古い snapshot が
// メールプロバイダーへ到達しないことを検証する。
func TestExportProfileDeletionGuard_DeleteWinsAfterNotificationSnapshot(t *testing.T) {
	t.Parallel()

	target := newCommittedExportDeletionTarget(t)
	ctx, cancel := context.WithTimeout(context.Background(), exportProfileDeletionRaceTimeout)
	defer cancel()

	exportRepo, notificationRepo, exportID, objectKey := newCommittedSucceededExport(t, target)
	guardRepo := repository.NewExportProfileDeletionGuardRepository(target.db)
	pausingGuard := newPausingExportProfileDeletionGuard(guardRepo)
	t.Cleanup(pausingGuard.unblock)
	storage := newFakeExportObjectStorage(t, target.profileID, exportID)
	storage.objects[objectKey] = []byte("archive")
	sender := &recordingExportCompletedSender{}

	sendUC := usecase.NewSendExportCompletedEmailUsecase(
		notificationRepo,
		pausingGuard,
		sender,
		exportPageURL,
	)
	deleteUC := usecase.NewDeleteExportsForProfileUsecase(exportRepo, notificationRepo, guardRepo, storage)

	sendResults := make(chan error, 1)
	go func() {
		sendResults <- sendUC.Execute(ctx, exportID)
	}()
	select {
	case <-pausingGuard.reached:
	case <-ctx.Done():
		t.Fatalf("通知 snapshot の読み取り後まで進まない: %v", ctx.Err())
	}

	if err := deleteUC.Execute(ctx, target.profileID); err != nil {
		t.Fatalf("Delete Execute() error = %v", err)
	}
	if got, err := exportRepo.FindByID(ctx, exportID); err != nil || got != nil {
		t.Errorf("Delete 後の FindByID() = (%v, %v), want (nil, nil)", got, err)
	}
	if got, err := notificationRepo.FindByExportID(ctx, exportID); err != nil || got != nil {
		t.Errorf("Delete 後の FindByExportID() = (%v, %v), want (nil, nil)", got, err)
	}
	if _, ok := storage.object(objectKey); ok {
		t.Error("Delete 後もオブジェクトが残っている")
	}

	pausingGuard.unblock()
	if err := receiveExportProfileDeletionRaceResult(t, "Send", sendResults); err != nil {
		t.Fatalf("Send Execute() error = %v", err)
	}
	if got := len(sender.sent); got != 0 {
		t.Errorf("送信件数 = %d, want 0", got)
	}
}
