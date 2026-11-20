package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// exportPageURLは完了メールが読み手を送るエクスポート画面。
const exportPageURL = "https://mewst.test/settings/export"

// sentExportCompletedEmailはsenderが求められた配信1件。
type sentExportCompletedEmail struct {
	to        string
	exportURL string
	locale    string
	exportID  string
}

// recordingExportCompletedSenderはメールプロバイダーの代役。配信をすべて記録し、
// メールが運ぶよう求められた宛先・言語・冪等キーの元をテストが読めるようにする。また、
// 失敗を返すことでプロバイダー障害でしか生じない状態を作れる。
type recordingExportCompletedSender struct {
	err error
	// beforeSendはこの配信がプロバイダーとやり取りしている間に実行される。同じ
	// エクスポートの別の配信がoutbox行を退役させ得るのはこの窓である。
	beforeSend func()
	sent       []sentExportCompletedEmail
}

func (s *recordingExportCompletedSender) Send(_ context.Context, to, exportURL, locale, exportID string) error {
	if s.beforeSend != nil {
		s.beforeSend()
	}
	s.sent = append(s.sent, sentExportCompletedEmail{
		to:        to,
		exportURL: exportURL,
		locale:    locale,
		exportID:  exportID,
	})
	return s.err
}

// recordingExportProfileDeletionGuardはすべての操作を許可し、問い合わせられた
// プロフィールを記録する。配信が入った削除境界をテストが読む手段になる。
type recordingExportProfileDeletionGuard struct {
	allowingExportProfileDeletionGuard
	// beforeBeginは共有lockに入る前に実行される。配信が待つのはこの窓であり、
	// この実行が読み終えたoutbox行を同じエクスポートの別の配信が退役させ得る。
	beforeBegin func()
	profileIDs  []model.ProfileID
}

func (g *recordingExportProfileDeletionGuard) BeginOperation(
	ctx context.Context,
	profileID model.ProfileID,
) (func() error, bool, error) {
	g.profileIDs = append(g.profileIDs, profileID)
	if g.beforeBegin != nil {
		g.beforeBegin()
	}
	return g.allowingExportProfileDeletionGuard.BeginOperation(ctx, profileID)
}

// retireExportCompletionNotificationは、別の配信のMarkSentと同じようにoutbox
// 行を削除する。並行する配信が残す状態を、テストがある実行に対して作れるようにする。
func retireExportCompletionNotification(t *testing.T, tx *sql.Tx, exportID model.ExportID) {
	t.Helper()

	if _, err := tx.Exec(
		"DELETE FROM export_completion_notifications WHERE export_id = $1",
		uuid.UUID(exportID),
	); err != nil {
		t.Fatalf("完了通知の退役に失敗: %v", err)
	}
}

// sendExportCompletedEmailFixtureは、テスト用トランザクションに配線した配信の
// UseCaseと、検証でoutboxを読み直すrepository、プロフィール削除guard、および
// メールプロバイダーの代役。
type sendExportCompletedEmailFixture struct {
	uc               *usecase.SendExportCompletedEmailUsecase
	notificationRepo *repository.ExportCompletionNotificationRepository
	guard            *recordingExportProfileDeletionGuard
	sender           *recordingExportCompletedSender
}

func newSendExportCompletedEmailFixture(t *testing.T, tx *sql.Tx) *sendExportCompletedEmailFixture {
	t.Helper()

	notificationRepo := repository.NewExportCompletionNotificationRepository(testutil.QueriesWithTx(tx))
	guard := &recordingExportProfileDeletionGuard{}
	sender := &recordingExportCompletedSender{}
	return &sendExportCompletedEmailFixture{
		uc: usecase.NewSendExportCompletedEmailUsecase(
			notificationRepo,
			guard,
			sender,
			exportPageURL,
		),
		notificationRepo: notificationRepo,
		guard:            guard,
		sender:           sender,
	}
}

// pendingは通知がまだ配信を待っているかどうかを返す。リコンシリエーションと
// ジョブキューが再試行するのはこれによる。
func (f *sendExportCompletedEmailFixture) pending(t *testing.T, exportID model.ExportID) bool {
	t.Helper()

	notification, err := f.notificationRepo.FindByExportID(context.Background(), exportID)
	if err != nil {
		t.Fatalf("FindByExportID()のエラー = %v", err)
	}
	return notification != nil
}

// TestSendExportCompletedEmailUsecase_Executeは配信1回が何を読み、何を残すかを
// 固定する。work intentは通知outboxであるため、知らせる対象のexport行はもう存在
// しなくてよく、配信が退役するのはプロバイダーが受け付けた後だけである。
func TestSendExportCompletedEmailUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("snapshotした宛先へ送信して送信待ち通知を退役させる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		exportID := testutil.NewExportBuilder(t, tx).
			WithProfileID(target.ProfileID).
			WithActorID(target.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			Build()

		// 宛先と言語はエクスポートが成功した時点のものであって、ユーザーが現在
		// 持つものではない。メールは取得し直した値ではなくsnapshotで求められる。
		const (
			recipientEmail = "snapshot@example.com"
			locale         = "en"
		)
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(exportID).
			WithActorID(target.ActorID).
			WithRecipientEmail(recipientEmail).
			WithLocale(locale).
			Build()

		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		want := sentExportCompletedEmail{
			to:        recipientEmail,
			exportURL: exportPageURL,
			locale:    locale,
			exportID:  exportID.String(),
		}
		if len(fixture.sender.sent) != 1 {
			t.Fatalf("送信件数 = %d、期待値 = 1", len(fixture.sender.sent))
		}
		if got := fixture.sender.sent[0]; got != want {
			t.Errorf("送信内容 = %+v、期待値 = %+v", got, want)
		}
		if fixture.pending(t, exportID) {
			t.Error("送信後も通知が送信待ちのまま残っている")
		}

		// 配信がguardするプロフィールは通知から得る。したがって削除境界の判断に、
		// 知らせる対象のexport行が存在する必要はない。
		if got := fixture.guard.profileIDs; len(got) != 1 || got[0] != target.ProfileID {
			t.Errorf("guardに渡されたprofile_id = %v、期待値 = [%v]", got, target.ProfileID)
		}
	})

	t.Run("cleanupがexport行を先に削除していてもsnapshotから送信する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		// 新しいエクスポートが成功し、この通知が知らせる行はcleanupにより既に
		// 削除されている。ここでメールを失わないためにoutboxが存在するので、配信は
		// export行に依存してはならない。
		exportID := model.ExportID(uuid.New())
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(exportID).
			WithActorID(target.ActorID).
			Build()

		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if len(fixture.sender.sent) != 1 {
			t.Fatalf("送信件数 = %d、期待値 = 1", len(fixture.sender.sent))
		}
		if fixture.pending(t, exportID) {
			t.Error("送信後も通知が送信待ちのまま残っている")
		}
	})

	t.Run("送信待ち通知が無ければ送信しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)

		// 行が無いのはメールが配信済みか、プロフィールの削除が取り消したかである。
		// 重複したジョブはここへ来るため、失敗を報告してもやることの無いジョブを
		// 再試行するだけになる。
		if err := fixture.uc.Execute(ctx, model.ExportID(uuid.New())); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if len(fixture.sender.sent) != 0 {
			t.Errorf("送信件数 = %d、期待値 = 0", len(fixture.sender.sent))
		}
	})

	t.Run("共有lockを待つ間に他の配信が退役させていたら送信しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		exportID := model.ExportID(uuid.New())
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(exportID).
			WithActorID(target.ActorID).
			Build()

		// この実行が共有lockを待つ間に、重複したジョブが同じ通知を読んで先に
		// 完了した。その待ち合わせの先でoutboxを読み直すことが、完了済みの処理に
		// ついてプロバイダーを呼ばないことを保証する。
		fixture.guard.beforeBegin = func() { retireExportCompletionNotification(t, tx, exportID) }

		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if len(fixture.sender.sent) != 0 {
			t.Errorf("送信件数 = %d、期待値 = 0", len(fixture.sender.sent))
		}
	})

	t.Run("送信中に他の配信が退役させていてもエラーにしない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		exportID := model.ExportID(uuid.New())
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(exportID).
			WithActorID(target.ActorID).
			Build()

		// この実行がプロバイダーとやり取りしている間に、別の配信が行を退役させた。
		// 2つの送信は並行して同じ冪等キーを運ぶため、Resendの24時間のウィンドウ内で
		// 重複排除される。この実行が退役させるものは残っていない。失敗を報告しても、
		// やることの無いジョブを再試行するだけになる。
		fixture.sender.beforeSend = func() { retireExportCompletionNotification(t, tx, exportID) }

		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if len(fixture.sender.sent) != 1 {
			t.Errorf("送信件数 = %d、期待値 = 1", len(fixture.sender.sent))
		}
		if fixture.pending(t, exportID) {
			t.Error("退役済みのはずの通知が送信待ちで残っている")
		}
	})

	t.Run("送信に失敗したら通知を残してエラーを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		exportID := model.ExportID(uuid.New())
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(exportID).
			WithActorID(target.ActorID).
			Build()

		// プロバイダーの障害は一時的なものなので、行はそれを越えて残る必要がある。
		// ジョブキューはその行から再試行し、試行を使い切った後はリコンシリエーションが
		// その行から再投入する。
		sendErr := errors.New("注入したメールプロバイダーのエラー")
		fixture.sender.err = sendErr

		err := fixture.uc.Execute(ctx, exportID)
		if !errors.Is(err, sendErr) {
			t.Fatalf("Execute()のエラー = %v、期待値 = %v", err, sendErr)
		}
		if !fixture.pending(t, exportID) {
			t.Error("送信に失敗したのに通知が退役している")
		}
	})

	t.Run("送信直後に退役できなかった通知は同じ冪等キーで再送される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		exportID := model.ExportID(uuid.New())
		add := func() {
			testutil.NewExportCompletionNotificationBuilder(t, tx).
				WithExportID(exportID).
				WithActorID(target.ActorID).
				WithRecipientEmail("snapshot@example.com").
				Build()
		}

		add()
		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		// 挿入し直した行は、送信と退役の間で終了したプロセスが残した行を表す。
		// 再試行は同じエクスポートIDを運ぶため、senderは同じ冪等キーを導出する。この
		// テストが固定するのはその入力であり、Resendが重複排除するのは24時間の冪等
		// ウィンドウ内に限られる。
		add()
		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("2回目のExecute()のエラー = %v", err)
		}

		if len(fixture.sender.sent) != 2 {
			t.Fatalf("送信件数 = %d、期待値 = 2", len(fixture.sender.sent))
		}
		if first, second := fixture.sender.sent[0], fixture.sender.sent[1]; first != second {
			t.Errorf("再送内容 = %+v、期待値 = %+v (同じ冪等キーで送られる必要がある)", second, first)
		}
		if fixture.pending(t, exportID) {
			t.Error("再送後も通知が送信待ちのまま残っている")
		}
	})
}
