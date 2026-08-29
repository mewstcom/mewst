package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// SendExportCompletedEmailTimeout bounds one delivery attempt. The run reads
// one row, calls the mail provider and writes one row, so this is the backstop
// for a provider that accepted the connection and then stopped answering: the
// job shares the default queue with the timeline delivery, and an attempt
// without a bound would hold one of those workers for as long as the process
// lives.
//
// [Ja] SendExportCompletedEmailTimeout は 1 回の配信試行の上限。実行は行を 1 件読み、
// メールプロバイダーを呼び、行を 1 件書くだけなので、これは接続を受け付けたまま応答
// しなくなったプロバイダーに対する歯止めである。このジョブはタイムライン配信と既定
// キューを共有しており、上限の無い試行はそれらの worker の 1 つを、プロセスが生きて
// いる限り占有する。
const SendExportCompletedEmailTimeout = 1 * time.Minute

// ExportCompletedSender sends the notification that an export is ready to
// download. The export ID is passed along with the message because the sender
// derives the delivery's idempotency key from it. A retry therefore reuses the
// same key; Resend suppresses duplicate sends with that key within its 24-hour
// idempotency window.
//
// [Ja] ExportCompletedSender はエクスポートがダウンロード可能になったことを知らせる
// 通知を送る。エクスポート ID をメッセージと一緒に渡すのは、sender が配信の冪等キーを
// そこから導出するためである。そのため再試行でも同じキーを使い、Resend の 24 時間の
// 冪等ウィンドウ内では、そのキーによる重複送信が抑止される。
type ExportCompletedSender interface {
	Send(ctx context.Context, to, exportURL, locale, exportID string) error
}

// SendExportCompletedEmailUsecase delivers one pending completion notification
// and retires it.
//
// The pending outbox row is the work intent, not the export: retention cleanup
// deletes the export row as soon as a newer export succeeds, and the profile,
// the recipient address and the locale this mail needs are snapshotted on the
// notification when the export succeeded. A run therefore neither reads the
// export nor cares whether it still exists.
//
// A run that delivers the mail but stops before retiring the row is retried
// from that row, and the delivery carries the same idempotency key. Within
// Resend's 24-hour idempotency window, the provider answers the retry with the
// first delivery instead of sending twice; a later retry can send again.
//
// [Ja] SendExportCompletedEmailUsecase は送信待ちの完了通知を 1 件配信し、退役させる。
//
// work intent は export ではなく送信待ちの outbox 行である。保持 cleanup は新しい
// エクスポートが成功した時点で export 行を削除し、このメールに必要なプロフィール・
// 宛先アドレス・locale はエクスポートの成功時に通知へ snapshot されている。したがって
// 実行は export を読まず、それがまだ存在するかも問わない。
//
// メールを配信した後、行を退役させる前に止まった実行は、その行から再試行される。配信は
// 同じ冪等キーを運び、Resend の 24 時間の冪等ウィンドウ内では、プロバイダーは 2 通目を
// 送らず最初の配信を返す。それより後の再試行では再送される可能性がある。
type SendExportCompletedEmailUsecase struct {
	notificationRepo *repository.ExportCompletionNotificationRepository
	deletionGuard    ExportProfileDeletionGuard
	sender           ExportCompletedSender
	exportPageURL    string
}

// NewSendExportCompletedEmailUsecase creates a SendExportCompletedEmailUsecase.
// exportPageURL is the absolute URL of the export screen, which is where the
// mail sends the reader instead of at the archive itself.
//
// [Ja] NewSendExportCompletedEmailUsecase は SendExportCompletedEmailUsecase を
// 生成する。exportPageURL はエクスポート画面の絶対 URL で、メールはアーカイブ自体では
// なくここへ読み手を送る。
func NewSendExportCompletedEmailUsecase(
	notificationRepo *repository.ExportCompletionNotificationRepository,
	deletionGuard ExportProfileDeletionGuard,
	sender ExportCompletedSender,
	exportPageURL string,
) *SendExportCompletedEmailUsecase {
	return &SendExportCompletedEmailUsecase{
		notificationRepo: notificationRepo,
		deletionGuard:    deletionGuard,
		sender:           sender,
		exportPageURL:    exportPageURL,
	}
}

// Execute delivers the export's completion notification if one is still
// pending.
//
// A missing notification finishes without error. The row is gone because the
// mail was already delivered or because a profile deletion cancelled it, and
// both mean this job carries no outstanding work; returning an error would only
// retry it until the attempts run out.
//
// [Ja] Execute はエクスポートの完了通知がまだ送信待ちであれば配信する。
//
// 通知が存在しない場合はエラーなしで完了する。行が無いのはメールが既に配信済みか、
// プロフィールの削除が取り消したかであり、どちらもこのジョブに未処理の作業が無いことを
// 意味する。エラーを返しても試行を使い切るまでリトライされるだけである。
func (uc *SendExportCompletedEmailUsecase) Execute(ctx context.Context, exportID model.ExportID) (err error) {
	notification, err := uc.notificationRepo.FindByExportID(ctx, exportID)
	if err != nil {
		return fmt.Errorf("送信待ちの完了通知の取得に失敗: %w", err)
	}
	if notification == nil {
		slog.InfoContext(ctx, "送信待ちの完了通知が無いためエクスポート完了メールを送信しません", "export_id", exportID.String())
		return nil
	}

	release, allowed, err := uc.deletionGuard.BeginOperation(ctx, notification.ProfileID)
	if err != nil {
		return fmt.Errorf("プロフィールの export 配信 guard の取得に失敗 (profile_id: %s): %w", notification.ProfileID.String(), err)
	}
	if !allowed {
		slog.InfoContext(ctx, "プロフィールが削除中または存在しないためエクスポート完了メールを送信しません",
			"export_id", exportID.String(),
			"profile_id", notification.ProfileID.String(),
		)
		return nil
	}
	defer func() {
		if releaseErr := release(); releaseErr != nil {
			err = errors.Join(err, fmt.Errorf("プロフィールの export 配信 guard の解放に失敗: %w", releaseErr))
		}
	}()

	// Another delivery of the same export can retire the notification while
	// this run waits for the shared lock. Read it again so a duplicate job does
	// not call the provider for work that is already done.
	//
	// [Ja] 共有 lock を待つ間に、同じエクスポートの別の配信が通知を退役させる場合が
	// ある。再取得することで、重複ジョブが完了済みの処理についてプロバイダーを
	// 呼ばないようにする。
	notification, err = uc.notificationRepo.FindByExportID(ctx, exportID)
	if err != nil {
		return fmt.Errorf("送信前の完了通知の再取得に失敗: %w", err)
	}
	if notification == nil {
		slog.InfoContext(ctx, "他の配信が完了通知を退役させたためエクスポート完了メールを送信しません",
			"export_id", exportID.String(),
		)
		return nil
	}

	// The recipient and the language come from the snapshot taken when the
	// export succeeded, so later account changes do not change this delivery.
	//
	// [Ja] 宛先と言語はエクスポート成功時の snapshot から得る。その後にユーザーが
	// アドレスや locale を変更しても、この配信には影響しない。
	if err := uc.sender.Send(ctx, notification.RecipientEmail, uc.exportPageURL, notification.Locale, exportID.String()); err != nil {
		return fmt.Errorf("エクスポート完了メールの送信に失敗 (export_id: %s): %w", exportID.String(), err)
	}

	sent, err := uc.notificationRepo.MarkSent(ctx, exportID)
	if err != nil {
		return fmt.Errorf("エクスポート完了通知の送信済み記録に失敗 (export_id: %s): %w", exportID.String(), err)
	}
	if !sent {
		// Another delivery retired the row while this one was with the provider.
		// The two sends overlap and carry the same idempotency key, so Resend
		// deduplicates them within its 24-hour window. There is nothing left for
		// this run to retire.
		//
		// [Ja] この実行がプロバイダーとやり取りしている間に、別の配信が行を退役させた。
		// 2 つの送信は並行して同じ冪等キーを運ぶため、Resend の 24 時間のウィンドウ内で
		// 重複排除される。この実行が退役させるものは残っていない。
		slog.WarnContext(ctx, "エクスポート完了通知は他の配信により既に退役済みでした", "export_id", exportID.String())
		return nil
	}

	slog.InfoContext(ctx, "エクスポート完了メールを送信しました", "export_id", exportID.String())
	return nil
}
