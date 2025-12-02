package model

import (
	"time"

	"github.com/google/uuid"
)

// Session はセッションのドメインモデル
type Session struct {
	ID         uuid.UUID
	ActorID    uuid.UUID
	Token      string
	IPAddress  string
	UserAgent  string
	SignedInAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
