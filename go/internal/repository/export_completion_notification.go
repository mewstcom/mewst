package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// ExportCompletionNotificationRepositoryは送信待ちの完了メールをexportの保持
// ライフサイクルから独立して保存する。
type ExportCompletionNotificationRepository struct {
	q *query.Queries
}

// NewExportCompletionNotificationRepositoryは
// ExportCompletionNotificationRepositoryを生成する。
func NewExportCompletionNotificationRepository(q *query.Queries) *ExportCompletionNotificationRepository {
	return &ExportCompletionNotificationRepository{q: q}
}

// WithTxはトランザクション内で動作するrepositoryを返す。
func (r *ExportCompletionNotificationRepository) WithTx(tx *sql.Tx) *ExportCompletionNotificationRepository {
	return &ExportCompletionNotificationRepository{q: r.q.WithTx(tx)}
}

// FindByExportIDはexportの送信待ち通知を返す。メール送信が不要になっている
// 場合は (nil, nil) を返す。保持cleanupによりexport行自体がすでに削除されて
// いる場合もある。
func (r *ExportCompletionNotificationRepository) FindByExportID(ctx context.Context, exportID model.ExportID) (*model.ExportCompletionNotification, error) {
	row, err := r.q.GetExportCompletionNotificationByExportID(ctx, uuid.UUID(exportID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return exportCompletionNotificationToModel(row), nil
}

// ExportCompletionNotificationCursorは (created_at, export_id) 順のページで
// 最後に取得した送信待ち通知を識別する。
type ExportCompletionNotificationCursor struct {
	CreatedAt time.Time
	ExportID  model.ExportID
}

// ListPendingはthresholdより前に作成された通知をcursorより後から古い順に
// 1ページ返し、次ページ用cursorも返す。nil cursorは最古の行から始め、次の
// cursorがnilなら走査は終端に達している。pageSizeは1以上である必要がある。
//
// 削除が始まったプロフィールの通知は返さない。配信はそのプロフィールの削除マーカーで
// 止まるため、再投入しても何も送らずに戻るジョブが生まれるだけである。これらの行を
// 消すのは親削除である。
func (r *ExportCompletionNotificationRepository) ListPending(
	ctx context.Context,
	threshold time.Time,
	cursor *ExportCompletionNotificationCursor,
	pageSize int32,
) ([]*model.ExportCompletionNotification, *ExportCompletionNotificationCursor, error) {
	afterTime := time.Time{}
	afterID := uuid.Nil
	if cursor != nil {
		afterTime = cursor.CreatedAt
		afterID = uuid.UUID(cursor.ExportID)
	}

	rows, err := r.q.ListPendingExportCompletionNotifications(ctx, query.ListPendingExportCompletionNotificationsParams{
		Threshold: threshold,
		AfterTime: afterTime,
		AfterID:   afterID,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, nil, err
	}

	notifications := make([]*model.ExportCompletionNotification, len(rows))
	for i, row := range rows {
		notifications[i] = exportCompletionNotificationToModel(row)
	}

	var next *ExportCompletionNotificationCursor
	if pageSize > 0 && len(notifications) == int(pageSize) {
		last := notifications[len(notifications)-1]
		next = &ExportCompletionNotificationCursor{
			CreatedAt: last.CreatedAt,
			ExportID:  last.ExportID,
		}
	}
	return notifications, next, nil
}

// MarkSentはメールを配信できた通知を退役させる。outbox行を削除し、同じ文で、
// export行がまだ存在する場合はそのlegacyなcompletion_notified_at列へ送信を
// mirrorする。通知を退役させたのがこの呼び出しかどうかを返す。falseは、別の配信
// またはプロフィールの削除が既に退役させたことを意味する。
func (r *ExportCompletionNotificationRepository) MarkSent(ctx context.Context, exportID model.ExportID) (bool, error) {
	n, err := r.q.MarkExportCompletionNotificationSent(ctx, uuid.UUID(exportID))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// DeleteByProfileIDはプロフィールの送信待ち通知をすべて取り消し、削除した行数を
// 返す。送信待ちのメールが不要になるのはプロフィールの削除であって、保持cleanupでは
// ない。cleanupがexport行を削除してもこれらに触れないのはそのためである。
func (r *ExportCompletionNotificationRepository) DeleteByProfileID(ctx context.Context, profileID model.ProfileID) (int64, error) {
	return r.q.DeleteExportCompletionNotificationsByProfileID(ctx, uuid.UUID(profileID))
}

func exportCompletionNotificationToModel(row query.ExportCompletionNotification) *model.ExportCompletionNotification {
	return &model.ExportCompletionNotification{
		ExportID:       model.ExportID(row.ExportID),
		ActorID:        model.ActorID(row.ActorID),
		ProfileID:      model.ProfileID(row.ProfileID),
		RecipientEmail: row.RecipientEmail,
		Locale:         row.Locale,
		CreatedAt:      row.CreatedAt,
	}
}
