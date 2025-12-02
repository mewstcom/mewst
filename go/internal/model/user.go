// Package model はドメインモデルを定義する
package model

import (
	"time"

	"github.com/google/uuid"
)

// User はユーザーのドメインモデル
type User struct {
	ID             uuid.UUID
	Email          string
	PasswordDigest string
	Locale         string
	TimeZone       string
	SignedUpAt     time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
