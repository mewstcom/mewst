// Package model はドメインモデルを定義する
package model

import "time"

// User はユーザーのドメインモデル
type User struct {
	ID             UserID
	Email          string
	PasswordDigest string
	Locale         string
	TimeZone       string
	SignedUpAt     time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
