package model

import "time"

// ExportCompletionNotification is a pending completion email. It snapshots the
// recipient and the profile the export belongs to when an export succeeds and
// survives deletion of the export row; deleting this row records that the email
// no longer needs to be retried.
//
// [Ja] ExportCompletionNotification は送信待ちの完了メール。export 成功時の宛先と、
// その export が属するプロフィールを snapshot して export 行の削除後も残り、この行の
// 削除がメールを再試行する必要がなくなったことを表す。
type ExportCompletionNotification struct {
	ExportID       ExportID
	ActorID        ActorID
	ProfileID      ProfileID
	RecipientEmail string
	Locale         string
	CreatedAt      time.Time
}
