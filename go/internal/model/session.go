package model

import (
	"time"
)

// Sessionはセッションのドメインモデル
type Session struct {
	ID         SessionID
	ActorID    ActorID
	Token      string
	IPAddress  string
	UserAgent  string
	SignedInAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
