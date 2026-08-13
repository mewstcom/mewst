package model

import "time"

// ExportStatus is the lifecycle status of an export. The four values and their
// names are kept in sync with the exports_status_check constraint and Wikino's
// export enum.
//
// [Ja] ExportStatus はエクスポートのライフサイクル状態。4 つの値と名前は
// exports_status_check 制約および Wikino の export enum と揃えている。
type ExportStatus string

const (
	// ExportStatusQueued means the export row is persisted as a durable work
	// intent and is waiting for the generation job to start.
	//
	// [Ja] ExportStatusQueued はエクスポート行が durable work intent として
	// 永続化され、生成ジョブの開始を待っている状態。
	ExportStatusQueued ExportStatus = "queued"

	// ExportStatusStarted means the generation job is running.
	//
	// [Ja] ExportStatusStarted は生成ジョブが実行中の状態。
	ExportStatusStarted ExportStatus = "started"

	// ExportStatusSucceeded means the zip was uploaded to R2 and is available
	// for download.
	//
	// [Ja] ExportStatusSucceeded は zip が R2 にアップロードされ、ダウンロード
	// 可能な状態。
	ExportStatusSucceeded ExportStatus = "succeeded"

	// ExportStatusFailed means the generation job reached its retry limit and
	// gave up.
	//
	// [Ja] ExportStatusFailed は生成ジョブがリトライ上限に達して諦めた状態。
	ExportStatusFailed ExportStatus = "failed"
)

// String returns the string representation of the ExportStatus.
//
// [Ja] String は ExportStatus の文字列表現を返す。
func (s ExportStatus) String() string { return string(s) }

// Export is the domain model for an export request. profile_id is the export
// target and drives the retention policy; actor_id is the requester used for
// the audit trail and for snapshotting the notification recipient on success.
//
// [Ja] Export はエクスポート依頼のドメインモデル。profile_id はエクスポート
// 対象で保持ポリシーを駆動し、actor_id は申請者として監査と成功時の通知先 snapshot
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
