package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

const (
	// ReconcileExportsTimeoutはリコンシリエーション1回の実行の上限。1回の
	// 実行の仕事量はページサイズと系統ごとの予算ですでに有界なので、これは通常の実行が
	// 近づく上限ではなく、前進しなくなった実行に対する歯止め。
	//
	// 値はリコンシリエーションの定期実行間隔より厳密に小さく保つ。このジョブはrunningを
	// 含む状態で一意なため、次の定期投入の時点でまだworkerを保持している実行があると
	// その投入がskipされ、停滞した実行がすでに遅らせた回復がもう1間隔分遅れる。間隔は
	// workerパッケージにあり、この関係は同パッケージのテストで固定する。
	ReconcileExportsTimeout = 4 * time.Minute

	// exportQueuedGraceは、リコンシリエーションが生成ジョブを再投入するまでに
	// queuedのエクスポートを放置する時間。ここで閉じる隙間はエクスポート行のcommitと
	// 直後の即時投入の間にあるため、その投入より長ければよい。再投入は一意ジョブに
	// より冪等なので、短すぎてもskipされる投入が1回増えるだけで済む。
	exportQueuedGrace = 1 * time.Minute

	// exportStartedGraceは、startedのエクスポートを停滞と見なすまでに生成の
	// timeoutへ上乗せする時間。timeoutはジョブキューが1回の試行に与える時間なので、
	// 猶予はworkerがtimeoutに気付いて結果を記録するまでを賄えばよい。これより早く
	// 回復させると、まだ実行を許されている試行と競合する。
	exportStartedGrace = 5 * time.Minute

	// exportNotificationGraceは、リコンシリエーションが完了通知メールを再投入する
	// まで送信待ちoutbox行を待つ時間。成功直後の投入・配信経路と回復経路が
	// 競合しないための猶予とする。
	exportNotificationGrace = 5 * time.Minute

	// defaultExportRecoveryPageSizeは1回の回復クエリが一度に保持する候補数。
	defaultExportRecoveryPageSize int32 = 100

	// defaultExportRecoveryBudgetは1回の実行が系統ごとに引き受ける候補数。
	defaultExportRecoveryBudget = 100
)

// ExportRecoveryLimitsはリコンシリエーション1回の実行を有界にする。PageSizeは
// 1クエリが保持する候補数を、Budgetは1回の実行が系統ごとに引き受ける候補数を制限
// する。これにより、バックログは1回の実行が全件を読み込んでジョブをバーストさせるの
// ではなく、複数回の実行で処理される。
//
// 2つを分けているのは、走査した候補と引き受けた候補が別物だからである。一意ジョブが
// すでに存在する候補は予算を使わずにskipされ、走査はcursorを進め続けるため、停滞
// した先頭候補が毎回の予算を独占することがない。
type ExportRecoveryLimits struct {
	PageSize int32
	Budget   int
}

// DefaultExportRecoveryLimitsは本番で使う上限値を返す。正でない値はこれで
// 置き換えるため、呼び出し側の設定によってリコンシリエーションが黙って何もしない
// 状態にはならない。
func DefaultExportRecoveryLimits() ExportRecoveryLimits {
	return ExportRecoveryLimits{
		PageSize: defaultExportRecoveryPageSize,
		Budget:   defaultExportRecoveryBudget,
	}
}

// normalizedは正でない値を既定値で置き換えた上限値を返す。
func (l ExportRecoveryLimits) normalized() ExportRecoveryLimits {
	defaults := DefaultExportRecoveryLimits()
	if l.PageSize <= 0 {
		l.PageSize = defaults.PageSize
	}
	if l.Budget <= 0 {
		l.Budget = defaults.Budget
	}
	return l
}

// ReconcileExportsUsecaseは通常の流れが取り残したエクスポートを収束させる。
// 生成ジョブが投入されないままcommitされたexport行、処理途中でプロセスが終了した
// 試行、投入されなかった後続処理などが対象。生成とcleanupはexportsから、完了
// メールは独立したoutboxから再導出するため、exportの保持処理が通知を消すことはない。
//
// 本UseCaseはDBの行を読んで動かし、一意ジョブを投入するだけである。オブジェクト
// ストレージには触れない。failedとして閉じたエクスポートはオブジェクトを手放し、その
// オブジェクトの回収は孤児回収の担当であるため。
//
// 本UseCaseが投入するジョブ種別は、その候補を生む状態にエクスポートが到達するより前に、
// 対応するWorkerが登録されている必要がある。Riverは未登録の種別のfetchにエラーを
// 返し、ジョブは試行を使い切ってdiscardedになる。discardedは一意性の状態集合の外に
// あるため、次回の実行が同じジョブを投入し直し、候補が存在する限りこの組が繰り返される。
type ReconcileExportsUsecase struct {
	exportRepo       *repository.ExportRepository
	notificationRepo *repository.ExportCompletionNotificationRepository
	dispatcher       *dispatcher.Dispatcher
	limits           ExportRecoveryLimits
}

// NewReconcileExportsUsecaseはReconcileExportsUsecaseを生成する。1回の実行を
// 別の上限で区切る理由がなければDefaultExportRecoveryLimits() を渡す。
func NewReconcileExportsUsecase(
	exportRepo *repository.ExportRepository,
	notificationRepo *repository.ExportCompletionNotificationRepository,
	d *dispatcher.Dispatcher,
	limits ExportRecoveryLimits,
) *ReconcileExportsUsecase {
	return &ReconcileExportsUsecase{
		exportRepo:       exportRepo,
		notificationRepo: notificationRepo,
		dispatcher:       d,
		limits:           limits.normalized(),
	}
}

// Executeは4系統の回復処理を走査し、そこで発生したエラーをまとめて返す。
//
// 前の系統が失敗しても、後続の系統は実行する。各系統は別々の行から別々の処理を回復
// するため、最初の失敗で止めると、1つの壊れた系統が他の系統の前進を隠してしまう。
// このジョブの試行回数は1回で、次回の定期実行が再試行にあたる。
func (uc *ReconcileExportsUsecase) Execute(ctx context.Context) error {
	// すべてのしきい値を1つの時刻から導く。実行に時間がかかっても、各系統が
	// 同じ時点を基準にするようにするため。
	now := time.Now()

	errs := []error{
		uc.requeueStaleQueued(ctx, now),
		uc.recoverStaleStarted(ctx, now),
		uc.notifyPendingCompletions(ctx, now),
		uc.cleanupProfilesWithOldSucceeded(ctx),
	}
	return errors.Join(errs...)
}

// requeueStaleQueuedは、猶予時間より長くqueuedのままのエクスポートについて
// 生成ジョブを再投入し、即時投入が失われたエクスポートを回復する。
func (uc *ReconcileExportsUsecase) requeueStaleQueued(ctx context.Context, now time.Time) error {
	threshold := now.Add(-exportQueuedGrace)

	taken, err := takeExportRecoveryCandidates(ctx, uc.limits,
		func(ctx context.Context, cursor *repository.ExportRecoveryCursor, pageSize int32) ([]*model.Export, *repository.ExportRecoveryCursor, error) {
			return uc.exportRepo.ListStaleQueued(ctx, threshold, cursor, pageSize)
		},
		func(ctx context.Context, export *model.Export) (bool, error) {
			return uc.enqueueGeneration(ctx, export)
		},
	)
	if taken > 0 {
		slog.InfoContext(ctx, "未投入のqueuedエクスポートを再投入しました", "count", taken)
	}
	if err != nil {
		return fmt.Errorf("queuedエクスポートの回復に失敗: %w", err)
	}
	return nil
}

// recoverStaleStartedは、試行が生成のtimeoutに猶予を足した時間より長く続いて
// いるエクスポートを収束させる。試行が残っているエクスポートはqueuedへ戻して再投入
// し、使い切ったものはfailedとして閉じる。これにより画面には、終わらない生成では
// なく、ユーザーが再実行できる失敗が表示される。
func (uc *ReconcileExportsUsecase) recoverStaleStarted(ctx context.Context, now time.Time) error {
	threshold := now.Add(-(GenerateExportTimeout + exportStartedGrace))

	taken, err := takeExportRecoveryCandidates(ctx, uc.limits,
		func(ctx context.Context, cursor *repository.ExportRecoveryCursor, pageSize int32) ([]*model.Export, *repository.ExportRecoveryCursor, error) {
			return uc.exportRepo.ListStaleStarted(ctx, threshold, cursor, pageSize)
		},
		func(ctx context.Context, export *model.Export) (bool, error) {
			if export.AttemptCount >= dispatcher.GenerateExportMaxAttempts {
				return uc.closeStaleStarted(ctx, export)
			}
			return uc.requeueStaleStarted(ctx, export)
		},
	)
	if taken > 0 {
		slog.InfoContext(ctx, "停滞したstartedエクスポートを回復しました", "count", taken)
	}
	if err != nil {
		return fmt.Errorf("startedエクスポートの回復に失敗: %w", err)
	}
	return nil
}

// requeueStaleStartedは停滞した試行をqueuedへ戻し、新しい生成ジョブを投入する。
// 遷移は行を読んだ時点のトークンでガードされるため、その間に競合する遷移が起きていた
// 場合は、現在その行を保持している側に委ねる。
func (uc *ReconcileExportsUsecase) requeueStaleStarted(ctx context.Context, export *model.Export) (bool, error) {
	requeued, err := uc.exportRepo.Requeue(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		return false, fmt.Errorf("停滞したエクスポートのqueuedへの差し戻しに失敗 (export_id: %s): %w", export.ID.String(), err)
	}
	if !requeued {
		return false, nil
	}

	slog.WarnContext(ctx, "停滞したエクスポートをqueuedへ戻しました",
		"export_id", export.ID.String(),
		"attempt_count", export.AttemptCount,
	)

	// ここで投入がskipされるのは、前の試行のジョブがまだ待機中または実行中の
	// 場合で、この行をqueuedへ戻したのはまさにその状況である。行はqueuedに
	// なっているため、そのジョブが消えた後にqueuedの系統が投入する。
	if _, err := uc.dispatcher.EnqueueGenerateExport(ctx, export.ID.String()); err != nil {
		return false, fmt.Errorf("差し戻したエクスポートの生成ジョブ投入に失敗 (export_id: %s): %w", export.ID.String(), err)
	}
	return true, nil
}

// closeStaleStartedは試行を使い切ったエクスポートを終わらせる。いずれかの試行が
// アップロードしたオブジェクトがあれば、この遷移によって手放され、孤児回収が回収する。
func (uc *ReconcileExportsUsecase) closeStaleStarted(ctx context.Context, export *model.Export) (bool, error) {
	failed, err := uc.exportRepo.MarkFailed(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		return false, fmt.Errorf("試行上限に達したエクスポートの失敗記録に失敗 (export_id: %s): %w", export.ID.String(), err)
	}
	if !failed {
		return false, nil
	}

	slog.WarnContext(ctx, "試行上限に達したエクスポートを失敗として終了しました",
		"export_id", export.ID.String(),
		"attempt_count", export.AttemptCount,
	)
	return true, nil
}

// notifyPendingCompletionsは猶予時間を超えてpendingのままの通知について、
// 完了メールジョブを投入する。
func (uc *ReconcileExportsUsecase) notifyPendingCompletions(ctx context.Context, now time.Time) error {
	threshold := now.Add(-exportNotificationGrace)

	taken, err := takeExportRecoveryCandidates(ctx, uc.limits,
		func(ctx context.Context, cursor *repository.ExportCompletionNotificationCursor, pageSize int32) ([]*model.ExportCompletionNotification, *repository.ExportCompletionNotificationCursor, error) {
			return uc.notificationRepo.ListPending(ctx, threshold, cursor, pageSize)
		},
		func(ctx context.Context, notification *model.ExportCompletionNotification) (bool, error) {
			inserted, err := uc.dispatcher.EnqueueSendExportCompletedEmail(ctx, notification.ExportID.String())
			if err != nil {
				return false, fmt.Errorf("完了通知メールジョブの投入に失敗 (export_id: %s): %w", notification.ExportID.String(), err)
			}
			return inserted, nil
		},
	)
	if taken > 0 {
		slog.InfoContext(ctx, "送信待ちのエクスポート完了通知を投入しました", "count", taken)
	}
	if err != nil {
		return fmt.Errorf("送信待ち完了通知の回復に失敗: %w", err)
	}
	return nil
}

// cleanupProfilesWithOldSucceededは、最新のものより古いsucceededの
// エクスポートをまだ持つプロフィールごとにcleanupジョブを投入し、成功後の投入が
// 失われた掃除を回復する。
func (uc *ReconcileExportsUsecase) cleanupProfilesWithOldSucceeded(ctx context.Context) error {
	taken, err := takeExportRecoveryCandidates(ctx, uc.limits,
		func(ctx context.Context, cursor *model.ProfileID, pageSize int32) ([]model.ProfileID, *model.ProfileID, error) {
			return uc.exportRepo.ListProfileIDsWithOldSucceeded(ctx, cursor, pageSize)
		},
		func(ctx context.Context, profileID model.ProfileID) (bool, error) {
			inserted, err := uc.dispatcher.EnqueueCleanupOldExports(ctx, profileID.String())
			if err != nil {
				return false, fmt.Errorf("旧エクスポート削除ジョブの投入に失敗 (profile_id: %s): %w", profileID.String(), err)
			}
			return inserted, nil
		},
	)
	if taken > 0 {
		slog.InfoContext(ctx, "旧エクスポートを持つプロフィールの削除ジョブを投入しました", "count", taken)
	}
	if err != nil {
		return fmt.Errorf("旧succeededエクスポートの回復に失敗: %w", err)
	}
	return nil
}

// enqueueGenerationはエクスポートの一意な生成ジョブを投入し、それを投入したのが
// 今回の実行かどうかを返す。
func (uc *ReconcileExportsUsecase) enqueueGeneration(ctx context.Context, export *model.Export) (bool, error) {
	inserted, err := uc.dispatcher.EnqueueGenerateExport(ctx, export.ID.String())
	if err != nil {
		return false, fmt.Errorf("生成ジョブの再投入に失敗 (export_id: %s): %w", export.ID.String(), err)
	}
	return inserted, nil
}

// takeExportRecoveryCandidatesは1系統の回復候補をページ単位で走査し、各候補を
// takeへ渡す。予算の分だけ候補を引き受けるか、走査が終端に達したところで止まる。
// 戻り値は引き受けた候補数。
//
// takeは、その候補が今回の実行の処理になったかどうかを返す。一意ジョブがすでに存在
// する候補や、ガード付き遷移が並行処理に負けた候補は数えず、予算も使わない。cursorは
// そうした候補を越えて進み続けるため、停滞した先頭候補の後ろにある候補にも同じ実行の
// 中で到達できる。
func takeExportRecoveryCandidates[Candidate any, Cursor any](
	ctx context.Context,
	limits ExportRecoveryLimits,
	list func(ctx context.Context, cursor *Cursor, pageSize int32) ([]Candidate, *Cursor, error),
	take func(ctx context.Context, candidate Candidate) (bool, error),
) (int, error) {
	var cursor *Cursor
	taken := 0

	for taken < limits.Budget {
		candidates, next, err := list(ctx, cursor, limits.PageSize)
		if err != nil {
			return taken, err
		}

		for _, candidate := range candidates {
			ok, err := take(ctx, candidate)
			if err != nil {
				return taken, err
			}
			if !ok {
				continue
			}
			taken++
			if taken >= limits.Budget {
				break
			}
		}

		if next == nil {
			break
		}
		cursor = next
	}

	return taken, nil
}
