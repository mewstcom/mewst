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

// ExportCompletionNotificationRepository stores pending completion emails
// independently of export retention.
//
// [Ja] ExportCompletionNotificationRepository は送信待ちの完了メールを export の保持
// ライフサイクルから独立して保存する。
type ExportCompletionNotificationRepository struct {
	q *query.Queries
}

// NewExportCompletionNotificationRepository creates an
// ExportCompletionNotificationRepository.
//
// [Ja] NewExportCompletionNotificationRepository は
// ExportCompletionNotificationRepository を生成する。
func NewExportCompletionNotificationRepository(q *query.Queries) *ExportCompletionNotificationRepository {
	return &ExportCompletionNotificationRepository{q: q}
}

// WithTx returns a repository that runs inside the transaction.
//
// [Ja] WithTx はトランザクション内で動作する repository を返す。
func (r *ExportCompletionNotificationRepository) WithTx(tx *sql.Tx) *ExportCompletionNotificationRepository {
	return &ExportCompletionNotificationRepository{q: r.q.WithTx(tx)}
}

// FindByExportID returns the pending notification for an export, or (nil, nil)
// when the email no longer needs to be sent. The export row itself may already
// have been removed by retention cleanup.
//
// [Ja] FindByExportID は export の送信待ち通知を返す。メール送信が不要になっている
// 場合は (nil, nil) を返す。保持 cleanup により export 行自体がすでに削除されて
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

// ExportCompletionNotificationCursor identifies the last pending notification
// in a page ordered by (created_at, export_id).
//
// [Ja] ExportCompletionNotificationCursor は (created_at, export_id) 順のページで
// 最後に取得した送信待ち通知を識別する。
type ExportCompletionNotificationCursor struct {
	CreatedAt time.Time
	ExportID  model.ExportID
}

// ListPending returns a page of notifications created before threshold, oldest
// first and strictly after cursor, with the cursor for the next page. A nil
// cursor starts at the oldest row, and a nil next cursor means the walk reached
// the end. pageSize must be at least 1.
//
// Notifications of a profile whose deletion has started are left out: delivery
// stops at that profile's deletion marker, so re-enqueueing them would only
// produce jobs that return without sending. Profile deletion is what removes
// those rows.
//
// [Ja] ListPending は threshold より前に作成された通知を cursor より後から古い順に
// 1 ページ返し、次ページ用 cursor も返す。nil cursor は最古の行から始め、次の
// cursor が nil なら走査は終端に達している。pageSize は 1 以上である必要がある。
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

// MarkSent retires a notification whose email was delivered: it deletes the
// outbox row and, in the same statement, mirrors the send into the export's
// legacy completion_notified_at column while that row still exists. It reports
// whether this call is the one that retired the notification; false means
// another delivery or a profile deletion already did.
//
// [Ja] MarkSent はメールを配信できた通知を退役させる。outbox 行を削除し、同じ文で、
// export 行がまだ存在する場合はその legacy な completion_notified_at 列へ送信を
// mirror する。通知を退役させたのがこの呼び出しかどうかを返す。false は、別の配信
// またはプロフィールの削除が既に退役させたことを意味する。
func (r *ExportCompletionNotificationRepository) MarkSent(ctx context.Context, exportID model.ExportID) (bool, error) {
	n, err := r.q.MarkExportCompletionNotificationSent(ctx, uuid.UUID(exportID))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// DeleteByProfileID cancels every pending notification of the profile and
// reports how many rows were deleted. Deleting a profile is what makes its
// pending emails unwanted; retention cleanup is not, which is why it deletes
// export rows without touching these.
//
// [Ja] DeleteByProfileID はプロフィールの送信待ち通知をすべて取り消し、削除した行数を
// 返す。送信待ちのメールが不要になるのはプロフィールの削除であって、保持 cleanup では
// ない。cleanup が export 行を削除してもこれらに触れないのはそのためである。
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
