package model

import (
	"time"
)

// UserProfile はユーザーとプロフィールの関連付けを表す
type UserProfile struct {
	ID        UserProfileID
	UserID    UserID
	ProfileID ProfileID
	CreatedAt time.Time
	UpdatedAt time.Time
}
