package model

import (
	"time"

	"github.com/google/uuid"
)

// Actor はアクターのドメインモデル
// ユーザーとプロフィールを関連付ける中間エンティティ
type Actor struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ProfileID uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}
