package model

import (
	"time"
)

// ProfileOwnerTypeUser is the owner type of a profile a user owns. It is the
// only owner type in use: a profile carrying it has a matching user_profiles
// row, and that association is what authorizes the user to act on the profile.
//
// [Ja] ProfileOwnerTypeUser はユーザーが所有するプロフィールの所有種別。現在
// 使われている所有種別はこれだけで、この値を持つプロフィールには対応する
// user_profiles 行があり、その関連付けがユーザーの操作を認可する根拠になる。
const ProfileOwnerTypeUser = "User"

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
