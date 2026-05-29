package model

import "github.com/google/uuid"

// UserID はユーザーのID型
type UserID uuid.UUID

// String はUserIDを文字列に変換する
func (id UserID) String() string { return uuid.UUID(id).String() }

// UserIDsToUUIDs はUserIDスライスをuuid.UUIDスライスに変換する
func UserIDsToUUIDs(ids []UserID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToUserIDs はuuid.UUIDスライスをUserIDスライスに変換する
func UUIDsToUserIDs(us []uuid.UUID) []UserID {
	ids := make([]UserID, len(us))
	for i, u := range us {
		ids[i] = UserID(u)
	}
	return ids
}

// ProfileID はプロフィールのID型
type ProfileID uuid.UUID

// String はProfileIDを文字列に変換する
func (id ProfileID) String() string { return uuid.UUID(id).String() }

// ProfileIDsToUUIDs はProfileIDスライスをuuid.UUIDスライスに変換する
func ProfileIDsToUUIDs(ids []ProfileID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToProfileIDs はuuid.UUIDスライスをProfileIDスライスに変換する
func UUIDsToProfileIDs(us []uuid.UUID) []ProfileID {
	ids := make([]ProfileID, len(us))
	for i, u := range us {
		ids[i] = ProfileID(u)
	}
	return ids
}

// ActorID はアクターのID型
type ActorID uuid.UUID

// String はActorIDを文字列に変換する
func (id ActorID) String() string { return uuid.UUID(id).String() }

// ActorIDsToUUIDs はActorIDスライスをuuid.UUIDスライスに変換する
func ActorIDsToUUIDs(ids []ActorID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToActorIDs はuuid.UUIDスライスをActorIDスライスに変換する
func UUIDsToActorIDs(us []uuid.UUID) []ActorID {
	ids := make([]ActorID, len(us))
	for i, u := range us {
		ids[i] = ActorID(u)
	}
	return ids
}

// SessionID はセッションのID型
type SessionID uuid.UUID

// String はSessionIDを文字列に変換する
func (id SessionID) String() string { return uuid.UUID(id).String() }

// SessionIDsToUUIDs はSessionIDスライスをuuid.UUIDスライスに変換する
func SessionIDsToUUIDs(ids []SessionID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToSessionIDs はuuid.UUIDスライスをSessionIDスライスに変換する
func UUIDsToSessionIDs(us []uuid.UUID) []SessionID {
	ids := make([]SessionID, len(us))
	for i, u := range us {
		ids[i] = SessionID(u)
	}
	return ids
}

// EmailConfirmationID はメール確認のID型
type EmailConfirmationID uuid.UUID

// String はEmailConfirmationIDを文字列に変換する
func (id EmailConfirmationID) String() string { return uuid.UUID(id).String() }

// EmailConfirmationIDsToUUIDs はEmailConfirmationIDスライスをuuid.UUIDスライスに変換する
func EmailConfirmationIDsToUUIDs(ids []EmailConfirmationID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToEmailConfirmationIDs はuuid.UUIDスライスをEmailConfirmationIDスライスに変換する
func UUIDsToEmailConfirmationIDs(us []uuid.UUID) []EmailConfirmationID {
	ids := make([]EmailConfirmationID, len(us))
	for i, u := range us {
		ids[i] = EmailConfirmationID(u)
	}
	return ids
}

// UserProfileID はユーザープロフィール関連付けのID型
type UserProfileID uuid.UUID

// String はUserProfileIDを文字列に変換する
func (id UserProfileID) String() string { return uuid.UUID(id).String() }

// UserProfileIDsToUUIDs はUserProfileIDスライスをuuid.UUIDスライスに変換する
func UserProfileIDsToUUIDs(ids []UserProfileID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToUserProfileIDs はuuid.UUIDスライスをUserProfileIDスライスに変換する
func UUIDsToUserProfileIDs(us []uuid.UUID) []UserProfileID {
	ids := make([]UserProfileID, len(us))
	for i, u := range us {
		ids[i] = UserProfileID(u)
	}
	return ids
}

// FeatureFlagID is the ID type for a feature flag.
// [Ja] FeatureFlagID はフィーチャーフラグの ID 型。
type FeatureFlagID uuid.UUID

// String returns the string representation of the FeatureFlagID.
// [Ja] String は FeatureFlagID の文字列表現を返す。
func (id FeatureFlagID) String() string { return uuid.UUID(id).String() }

// FeatureFlagName is the type for a feature flag name.
// [Ja] FeatureFlagName はフィーチャーフラグ名の型。
type FeatureFlagName string

// String returns the string representation of the FeatureFlagName.
// [Ja] String は FeatureFlagName の文字列表現を返す。
func (n FeatureFlagName) String() string { return string(n) }

// PostID is the ID type for a post.
// [Ja] PostID は投稿の ID 型。
type PostID uuid.UUID

// String returns the string representation of the PostID.
// [Ja] String は PostID の文字列表現を返す。
func (id PostID) String() string { return uuid.UUID(id).String() }

// OauthApplicationID is the ID type for an OAuth application.
// [Ja] OauthApplicationID は OAuth アプリケーションの ID 型。
type OauthApplicationID uuid.UUID

// String returns the string representation of the OauthApplicationID.
// [Ja] String は OauthApplicationID の文字列表現を返す。
func (id OauthApplicationID) String() string { return uuid.UUID(id).String() }

// HomeTimelinePostID is the ID type for a home timeline post.
// [Ja] HomeTimelinePostID はホームタイムライン投稿の ID 型。
type HomeTimelinePostID uuid.UUID

// String returns the string representation of the HomeTimelinePostID.
// [Ja] String は HomeTimelinePostID の文字列表現を返す。
func (id HomeTimelinePostID) String() string { return uuid.UUID(id).String() }

// FollowID is the ID type for a follow.
// [Ja] FollowID はフォローの ID 型。
type FollowID uuid.UUID

// String returns the string representation of the FollowID.
// [Ja] String は FollowID の文字列表現を返す。
func (id FollowID) String() string { return uuid.UUID(id).String() }
