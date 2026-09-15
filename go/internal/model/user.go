// Package modelはドメインモデルを定義する
package model

import "time"

// Userはユーザーのドメインモデル
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
