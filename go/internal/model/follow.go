package model

import "time"

// Follow is the domain model for a follow relationship. The source profile
// follows the target profile, so a post author's followers are the source
// profiles of the follows whose target is the author.
//
// [Ja] Follow はフォロー関係のドメインモデル。source プロフィールが target
// プロフィールをフォローする関係を表し、投稿者のフォロワーは target が投稿者で
// ある follow の source プロフィール群にあたる。
type Follow struct {
	ID              FollowID
	SourceProfileID ProfileID
	TargetProfileID ProfileID
	FollowedAt      time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
