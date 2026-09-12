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

// exportPageURL is the export screen the completion mail sends its reader to.
//
// [Ja] exportPageURL は完了メールが読み手を送るエクスポート画面。
const exportPageURL = "https://mewst.test/settings/export"

// sentExportCompletedEmail is one delivery the sender was asked for.
//
// [Ja] sentExportCompletedEmail は sender が求められた配信 1 件。
type sentExportCompletedEmail struct {
	to        string
	exportURL string
	locale    string
	exportID  string
}

// recordingExportCompletedSender stands in for the mail provider. It records
// every delivery so a test can read the address, language and idempotency
// source the mail was asked to carry, and it can fail to put the caller in the
// state a provider outage produces.
//
// [Ja] recordingExportCompletedSender はメールプロバイダーの代役。配信をすべて記録し、
// メールが運ぶよう求められた宛先・言語・冪等キーの元をテストが読めるようにする。また、
// 失敗を返すことでプロバイダー障害でしか生じない状態を作れる。
type recordingExportCompletedSender struct {
	err error
	// beforeSend runs while this delivery is with the provider, which is the
	// window another delivery of the same export can retire the outbox row in.
	//
	// [Ja] beforeSend はこの配信がプロバイダーとやり取りしている間に実行される。同じ
	// エクスポートの別の配信が outbox 行を退役させ得るのはこの窓である。
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

// recordingExportProfileDeletionGuard allows every operation and records the
// profile it was asked about, which is how a test reads the deletion boundary
// the delivery decided to enter.
//
// [Ja] recordingExportProfileDeletionGuard はすべての操作を許可し、問い合わせられた
// プロフィールを記録する。配信が入った削除境界をテストが読む手段になる。
type recordingExportProfileDeletionGuard struct {
	allowingExportProfileDeletionGuard
	// beforeBegin runs before the shared lock is entered, which is the window a
	// delivery waits in and another delivery of the same export can retire the
	// outbox row this one already read.
	//
	// [Ja] beforeBegin は共有 lock に入る前に実行される。配信が待つのはこの窓であり、
	// この実行が読み終えた outbox 行を同じエクスポートの別の配信が退役させ得る。
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

// retireExportCompletionNotification removes the outbox row the way another
// delivery's MarkSent does, so a test can put one run in the state a concurrent
// delivery leaves behind.
//
// [Ja] retireExportCompletionNotification は、別の配信の MarkSent と同じように outbox
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

// sendExportCompletedEmailFixture is the delivery UseCase wired to a test
// transaction, together with the repository the assertions read the outbox back
// with, the profile deletion guard and the mail provider stand-in.
//
// [Ja] sendExportCompletedEmailFixture は、テスト用トランザクションに配線した配信の
// UseCase と、検証で outbox を読み直す repository、プロフィール削除 guard、および
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

// pending reports whether the notification is still waiting to be delivered,
// which is what makes reconciliation and the job queue try again.
//
// [Ja] pending は通知がまだ配信を待っているかどうかを返す。リコンシリエーションと
// ジョブキューが再試行するのはこれによる。
func (f *sendExportCompletedEmailFixture) pending(t *testing.T, exportID model.ExportID) bool {
	t.Helper()

	notification, err := f.notificationRepo.FindByExportID(context.Background(), exportID)
	if err != nil {
		t.Fatalf("FindByExportID() error = %v", err)
	}
	return notification != nil
}

// TestSendExportCompletedEmailUsecase_Execute pins what one delivery reads and
// what it leaves behind. The notification outbox is the work intent, so the
// export row it announces does not have to exist any more, and a delivery is
// only retired once the provider accepted it.
//
// [Ja] TestSendExportCompletedEmailUsecase_Execute は配信 1 回が何を読み、何を残すかを
// 固定する。work intent は通知 outbox であるため、知らせる対象の export 行はもう存在
// しなくてよく、配信が退役するのはプロバイダーが受け付けた後だけである。
func TestSendExportCompletedEmailUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("snapshot した宛先へ送信して送信待ち通知を退役させる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		exportID := testutil.NewExportBuilder(t, tx).
			WithProfileID(target.ProfileID).
			WithActorID(target.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			Build()

		// The address and the language are the ones the export succeeded with,
		// not the ones the user holds now, so the mail is asked for with the
		// snapshot rather than with a fresh lookup.
		//
		// [Ja] 宛先と言語はエクスポートが成功した時点のものであって、ユーザーが現在
		// 持つものではない。メールは取得し直した値ではなく snapshot で求められる。
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
			t.Fatalf("Execute() error = %v", err)
		}

		want := sentExportCompletedEmail{
			to:        recipientEmail,
			exportURL: exportPageURL,
			locale:    locale,
			exportID:  exportID.String(),
		}
		if len(fixture.sender.sent) != 1 {
			t.Fatalf("送信件数 = %d, want 1", len(fixture.sender.sent))
		}
		if got := fixture.sender.sent[0]; got != want {
			t.Errorf("送信内容 = %+v, want %+v", got, want)
		}
		if fixture.pending(t, exportID) {
			t.Error("送信後も通知が送信待ちのまま残っている")
		}

		// The profile the delivery guards against comes from the notification, so
		// the export row it announces does not have to exist for the deletion
		// boundary to be decided.
		//
		// [Ja] 配信が guard するプロフィールは通知から得る。したがって削除境界の判断に、
		// 知らせる対象の export 行が存在する必要はない。
		if got := fixture.guard.profileIDs; len(got) != 1 || got[0] != target.ProfileID {
			t.Errorf("guard に渡された profile_id = %v, want [%v]", got, target.ProfileID)
		}
	})

	t.Run("cleanup が export 行を先に削除していても snapshot から送信する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		// A newer export succeeded and the cleanup already removed the row this
		// notification announces. Losing the mail here is what the outbox exists
		// to prevent, so the delivery must not depend on the export row.
		//
		// [Ja] 新しいエクスポートが成功し、この通知が知らせる行は cleanup により既に
		// 削除されている。ここでメールを失わないために outbox が存在するので、配信は
		// export 行に依存してはならない。
		exportID := model.ExportID(uuid.New())
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(exportID).
			WithActorID(target.ActorID).
			Build()

		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if len(fixture.sender.sent) != 1 {
			t.Fatalf("送信件数 = %d, want 1", len(fixture.sender.sent))
		}
		if fixture.pending(t, exportID) {
			t.Error("送信後も通知が送信待ちのまま残っている")
		}
	})

	t.Run("送信待ち通知が無ければ送信しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)

		// The row is gone because the mail was delivered or because a profile
		// deletion cancelled it. A duplicate job lands here, and reporting a
		// failure would only retry a job with nothing left to do.
		//
		// [Ja] 行が無いのはメールが配信済みか、プロフィールの削除が取り消したかである。
		// 重複したジョブはここへ来るため、失敗を報告してもやることの無いジョブを
		// 再試行するだけになる。
		if err := fixture.uc.Execute(ctx, model.ExportID(uuid.New())); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if len(fixture.sender.sent) != 0 {
			t.Errorf("送信件数 = %d, want 0", len(fixture.sender.sent))
		}
	})

	t.Run("共有 lock を待つ間に他の配信が退役させていたら送信しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newSendExportCompletedEmailFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		exportID := model.ExportID(uuid.New())
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(exportID).
			WithActorID(target.ActorID).
			Build()

		// A duplicate job read the same notification and finished first while this
		// run was waiting for the shared lock. Reading the outbox again on the
		// other side of that wait is what keeps the provider from being called for
		// work that is already done.
		//
		// [Ja] この実行が共有 lock を待つ間に、重複したジョブが同じ通知を読んで先に
		// 完了した。その待ち合わせの先で outbox を読み直すことが、完了済みの処理に
		// ついてプロバイダーを呼ばないことを保証する。
		fixture.guard.beforeBegin = func() { retireExportCompletionNotification(t, tx, exportID) }

		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if len(fixture.sender.sent) != 0 {
			t.Errorf("送信件数 = %d, want 0", len(fixture.sender.sent))
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

		// Another delivery retired the row while this one was with the provider.
		// The two sends overlap and carry the same idempotency key, so Resend
		// deduplicates them within its 24-hour window. This run has nothing left to
		// retire; reporting a failure would only retry a job with no work in it.
		//
		// [Ja] この実行がプロバイダーとやり取りしている間に、別の配信が行を退役させた。
		// 2 つの送信は並行して同じ冪等キーを運ぶため、Resend の 24 時間のウィンドウ内で
		// 重複排除される。この実行が退役させるものは残っていない。失敗を報告しても、
		// やることの無いジョブを再試行するだけになる。
		fixture.sender.beforeSend = func() { retireExportCompletionNotification(t, tx, exportID) }

		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if len(fixture.sender.sent) != 1 {
			t.Errorf("送信件数 = %d, want 1", len(fixture.sender.sent))
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

		// A provider outage is temporary, so the row has to survive it: the job
		// queue retries from it, and the reconciliation re-enqueues from it once
		// the attempts are used up.
		//
		// [Ja] プロバイダーの障害は一時的なものなので、行はそれを越えて残る必要がある。
		// ジョブキューはその行から再試行し、試行を使い切った後はリコンシリエーションが
		// その行から再投入する。
		sendErr := errors.New("注入したメールプロバイダーのエラー")
		fixture.sender.err = sendErr

		err := fixture.uc.Execute(ctx, exportID)
		if !errors.Is(err, sendErr) {
			t.Fatalf("Execute() error = %v, want %v", err, sendErr)
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
			t.Fatalf("Execute() error = %v", err)
		}

		// The re-inserted row stands for the one a process that died between the
		// send and the retire left behind. The retry carries the same export ID, so
		// the sender derives the same idempotency key. This test pins that input;
		// Resend deduplicates it only within its 24-hour idempotency window.
		//
		// [Ja] 挿入し直した行は、送信と退役の間で終了したプロセスが残した行を表す。
		// 再試行は同じエクスポート ID を運ぶため、sender は同じ冪等キーを導出する。この
		// テストが固定するのはその入力であり、Resend が重複排除するのは 24 時間の冪等
		// ウィンドウ内に限られる。
		add()
		if err := fixture.uc.Execute(ctx, exportID); err != nil {
			t.Fatalf("2 回目の Execute() error = %v", err)
		}

		if len(fixture.sender.sent) != 2 {
			t.Fatalf("送信件数 = %d, want 2", len(fixture.sender.sent))
		}
		if first, second := fixture.sender.sent[0], fixture.sender.sent[1]; first != second {
			t.Errorf("再送内容 = %+v, want %+v (同じ冪等キーで送られる必要がある)", second, first)
		}
		if fixture.pending(t, exportID) {
			t.Error("再送後も通知が送信待ちのまま残っている")
		}
	})
}
