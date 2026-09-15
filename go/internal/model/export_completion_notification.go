package model

import "time"

// ExportCompletionNotificationは送信待ちの完了メール。export成功時の宛先と、
// そのexportが属するプロフィールをsnapshotしてexport行の削除後も残り、この行の
// 削除がメールを再試行する必要がなくなったことを表す。
type ExportCompletionNotification struct {
	ExportID       ExportID
	ActorID        ActorID
	ProfileID      ProfileID
	RecipientEmail string
	Locale         string
	CreatedAt      time.Time
}
