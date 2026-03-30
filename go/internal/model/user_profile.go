package model

import (
	"time"

	"github.com/google/uuid"
)

// UserProfile はユーザーとプロフィールの関連付けを表す
type UserProfile struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ProfileID uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}
