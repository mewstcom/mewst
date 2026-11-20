package model

import (
	"time"
)

// ProfileOwnerTypeUserはユーザーが所有するプロフィールの所有種別。現在
// 使われている所有種別はこれだけで、この値を持つプロフィールには対応する
// user_profiles行があり、その関連付けがユーザーの操作を認可する根拠になる。
const ProfileOwnerTypeUser = "User"

// Profileはプロフィールを表す
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
