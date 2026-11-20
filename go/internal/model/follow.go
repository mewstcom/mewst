package model

import "time"

// Followはフォロー関係のドメインモデル。sourceプロフィールがtarget
// プロフィールをフォローする関係を表し、投稿者のフォロワーはtargetが投稿者で
// あるfollowのsourceプロフィール群にあたる。
type Follow struct {
	ID              FollowID
	SourceProfileID ProfileID
	TargetProfileID ProfileID
	FollowedAt      time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
