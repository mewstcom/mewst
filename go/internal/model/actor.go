package model

import (
	"time"
)

// Actor はアクターのドメインモデル
// ユーザーとプロフィールを関連付ける中間エンティティ
type Actor struct {
	ID        ActorID
	UserID    UserID
	ProfileID ProfileID
	CreatedAt time.Time
	UpdatedAt time.Time
}
