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

// SendExportCompletedEmailTimeoutは1回の配信試行の上限。実行は行を1件読み、
// メールプロバイダーを呼び、行を1件書くだけなので、これは接続を受け付けたまま応答
// しなくなったプロバイダーに対する歯止めである。このジョブはタイムライン配信と既定
// キューを共有しており、上限の無い試行はそれらのworkerの1つを、プロセスが生きて
// いる限り占有する。
const SendExportCompletedEmailTimeout = 1 * time.Minute

// ExportCompletedSenderはエクスポートがダウンロード可能になったことを知らせる
// 通知を送る。エクスポートIDをメッセージと一緒に渡すのは、senderが配信の冪等キーを
// そこから導出するためである。そのため再試行でも同じキーを使い、Resendの24時間の
// 冪等ウィンドウ内では、そのキーによる重複送信が抑止される。
type ExportCompletedSender interface {
	Send(ctx context.Context, to, exportURL, locale, exportID string) error
}

// SendExportCompletedEmailUsecaseは送信待ちの完了通知を1件配信し、退役させる。
//
// work intentはexportではなく送信待ちのoutbox行である。保持cleanupは新しい
// エクスポートが成功した時点でexport行を削除し、このメールに必要なプロフィール・
// 宛先アドレス・localeはエクスポートの成功時に通知へsnapshotされている。したがって
// 実行はexportを読まず、それがまだ存在するかも問わない。
//
// メールを配信した後、行を退役させる前に止まった実行は、その行から再試行される。配信は
// 同じ冪等キーを運び、Resendの24時間の冪等ウィンドウ内では、プロバイダーは2通目を
// 送らず最初の配信を返す。それより後の再試行では再送される可能性がある。
type SendExportCompletedEmailUsecase struct {
	notificationRepo *repository.ExportCompletionNotificationRepository
	deletionGuard    ExportProfileDeletionGuard
	sender           ExportCompletedSender
	exportPageURL    string
}

// NewSendExportCompletedEmailUsecaseはSendExportCompletedEmailUsecaseを
// 生成する。exportPageURLはエクスポート画面の絶対URLで、メールはアーカイブ自体では
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

// Executeはエクスポートの完了通知がまだ送信待ちであれば配信する。
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
		return fmt.Errorf("プロフィールのexport配信guardの取得に失敗 (profile_id: %s): %w", notification.ProfileID.String(), err)
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
			err = errors.Join(err, fmt.Errorf("プロフィールのexport配信guardの解放に失敗: %w", releaseErr))
		}
	}()

	// 共有lockを待つ間に、同じエクスポートの別の配信が通知を退役させる場合が
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

	// 宛先と言語はエクスポート成功時のsnapshotから得る。その後にユーザーが
	// アドレスやlocaleを変更しても、この配信には影響しない。
	if err := uc.sender.Send(ctx, notification.RecipientEmail, uc.exportPageURL, notification.Locale, exportID.String()); err != nil {
		return fmt.Errorf("エクスポート完了メールの送信に失敗 (export_id: %s): %w", exportID.String(), err)
	}

	sent, err := uc.notificationRepo.MarkSent(ctx, exportID)
	if err != nil {
		return fmt.Errorf("エクスポート完了通知の送信済み記録に失敗 (export_id: %s): %w", exportID.String(), err)
	}
	if !sent {
		// この実行がプロバイダーとやり取りしている間に、別の配信が行を退役させた。
		// 2つの送信は並行して同じ冪等キーを運ぶため、Resendの24時間のウィンドウ内で
		// 重複排除される。この実行が退役させるものは残っていない。
		slog.WarnContext(ctx, "エクスポート完了通知は他の配信により既に退役済みでした", "export_id", exportID.String())
		return nil
	}

	slog.InfoContext(ctx, "エクスポート完了メールを送信しました", "export_id", exportID.String())
	return nil
}
