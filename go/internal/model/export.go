package model

import "time"

// ExportStatusはエクスポートのライフサイクル状態。4つの値と名前は
// exports_status_check制約およびWikinoのexport enumと揃えている。
type ExportStatus string

const (
	// ExportStatusQueuedはエクスポート行がdurable work intentとして
	// 永続化され、生成ジョブの開始を待っている状態。
	ExportStatusQueued ExportStatus = "queued"

	// ExportStatusStartedは生成ジョブが実行中の状態。
	ExportStatusStarted ExportStatus = "started"

	// ExportStatusSucceededはzipがR2にアップロードされ、ダウンロード
	// 可能な状態。
	ExportStatusSucceeded ExportStatus = "succeeded"

	// ExportStatusFailedは生成ジョブがリトライ上限に達して諦めた状態。
	ExportStatusFailed ExportStatus = "failed"
)

// StringはExportStatusの文字列表現を返す。
func (s ExportStatus) String() string { return string(s) }

// Exportはエクスポート依頼のドメインモデル。profile_idはエクスポート
// 対象で保持ポリシーを駆動し、actor_idは申請者として監査と成功時の通知先snapshot
// の解決に使う。
type Export struct {
	ID           ExportID
	ProfileID    ProfileID
	ActorID      ActorID
	Status       ExportStatus
	ObjectKey    *string
	AttemptCount int32
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
