package model

import (
	"time"
)

// Profile はプロフィールを表す
type Profile struct {
	ID            ProfileID
	OwnerType     string
	Atname        string
	Name          string
	Description   string
	ImageURL      string
	JoinedAt      time.Time
	AvatarKind    string
	GravatarEmail string
	GravatarURL   string
	DiscardedAt   *time.Time
	LastPostAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
