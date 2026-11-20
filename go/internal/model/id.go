package model

import "github.com/google/uuid"

// UserIDはユーザーのID型
type UserID uuid.UUID

// StringはUserIDを文字列に変換する
func (id UserID) String() string { return uuid.UUID(id).String() }

// UserIDsToUUIDsはUserIDスライスをuuid.UUIDスライスに変換する
func UserIDsToUUIDs(ids []UserID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToUserIDsはuuid.UUIDスライスをUserIDスライスに変換する
func UUIDsToUserIDs(us []uuid.UUID) []UserID {
	ids := make([]UserID, len(us))
	for i, u := range us {
		ids[i] = UserID(u)
	}
	return ids
}

// ProfileIDはプロフィールのID型
type ProfileID uuid.UUID

// StringはProfileIDを文字列に変換する
func (id ProfileID) String() string { return uuid.UUID(id).String() }

// ProfileIDsToUUIDsはProfileIDスライスをuuid.UUIDスライスに変換する
func ProfileIDsToUUIDs(ids []ProfileID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToProfileIDsはuuid.UUIDスライスをProfileIDスライスに変換する
func UUIDsToProfileIDs(us []uuid.UUID) []ProfileID {
	ids := make([]ProfileID, len(us))
	for i, u := range us {
		ids[i] = ProfileID(u)
	}
	return ids
}

// ActorIDはアクターのID型
type ActorID uuid.UUID

// StringはActorIDを文字列に変換する
func (id ActorID) String() string { return uuid.UUID(id).String() }

// ActorIDsToUUIDsはActorIDスライスをuuid.UUIDスライスに変換する
func ActorIDsToUUIDs(ids []ActorID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToActorIDsはuuid.UUIDスライスをActorIDスライスに変換する
func UUIDsToActorIDs(us []uuid.UUID) []ActorID {
	ids := make([]ActorID, len(us))
	for i, u := range us {
		ids[i] = ActorID(u)
	}
	return ids
}

// SessionIDはセッションのID型
type SessionID uuid.UUID

// StringはSessionIDを文字列に変換する
func (id SessionID) String() string { return uuid.UUID(id).String() }

// SessionIDsToUUIDsはSessionIDスライスをuuid.UUIDスライスに変換する
func SessionIDsToUUIDs(ids []SessionID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToSessionIDsはuuid.UUIDスライスをSessionIDスライスに変換する
func UUIDsToSessionIDs(us []uuid.UUID) []SessionID {
	ids := make([]SessionID, len(us))
	for i, u := range us {
		ids[i] = SessionID(u)
	}
	return ids
}

// EmailConfirmationIDはメール確認のID型
type EmailConfirmationID uuid.UUID

// StringはEmailConfirmationIDを文字列に変換する
func (id EmailConfirmationID) String() string { return uuid.UUID(id).String() }

// EmailConfirmationIDsToUUIDsはEmailConfirmationIDスライスをuuid.UUIDスライスに変換する
func EmailConfirmationIDsToUUIDs(ids []EmailConfirmationID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToEmailConfirmationIDsはuuid.UUIDスライスをEmailConfirmationIDスライスに変換する
func UUIDsToEmailConfirmationIDs(us []uuid.UUID) []EmailConfirmationID {
	ids := make([]EmailConfirmationID, len(us))
	for i, u := range us {
		ids[i] = EmailConfirmationID(u)
	}
	return ids
}

// UserProfileIDはユーザープロフィール関連付けのID型
type UserProfileID uuid.UUID

// StringはUserProfileIDを文字列に変換する
func (id UserProfileID) String() string { return uuid.UUID(id).String() }

// UserProfileIDsToUUIDsはUserProfileIDスライスをuuid.UUIDスライスに変換する
func UserProfileIDsToUUIDs(ids []UserProfileID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToUserProfileIDsはuuid.UUIDスライスをUserProfileIDスライスに変換する
func UUIDsToUserProfileIDs(us []uuid.UUID) []UserProfileID {
	ids := make([]UserProfileID, len(us))
	for i, u := range us {
		ids[i] = UserProfileID(u)
	}
	return ids
}

// FeatureFlagIDはフィーチャーフラグのID型。
type FeatureFlagID uuid.UUID

// StringはFeatureFlagIDの文字列表現を返す。
func (id FeatureFlagID) String() string { return uuid.UUID(id).String() }

// FeatureFlagNameはフィーチャーフラグ名の型。
type FeatureFlagName string

// StringはFeatureFlagNameの文字列表現を返す。
func (n FeatureFlagName) String() string { return string(n) }

// PostIDは投稿のID型。
type PostID uuid.UUID

// StringはPostIDの文字列表現を返す。
func (id PostID) String() string { return uuid.UUID(id).String() }

// OauthApplicationIDはOAuthアプリケーションのID型。
type OauthApplicationID uuid.UUID

// StringはOauthApplicationIDの文字列表現を返す。
func (id OauthApplicationID) String() string { return uuid.UUID(id).String() }

// HomeTimelinePostIDはホームタイムライン投稿のID型。
type HomeTimelinePostID uuid.UUID

// StringはHomeTimelinePostIDの文字列表現を返す。
func (id HomeTimelinePostID) String() string { return uuid.UUID(id).String() }

// FollowIDはフォローのID型。
type FollowID uuid.UUID

// StringはFollowIDの文字列表現を返す。
func (id FollowID) String() string { return uuid.UUID(id).String() }

// LinkIDはリンクのID型。
type LinkID uuid.UUID

// StringはLinkIDの文字列表現を返す。
func (id LinkID) String() string { return uuid.UUID(id).String() }

// PostLinkIDは投稿とリンクの関連付けのID型。
type PostLinkID uuid.UUID

// StringはPostLinkIDの文字列表現を返す。
func (id PostLinkID) String() string { return uuid.UUID(id).String() }

// ExportIDはエクスポートのID型。
type ExportID uuid.UUID

// StringはExportIDの文字列表現を返す。
func (id ExportID) String() string { return uuid.UUID(id).String() }

// ExportIDsToUUIDsはExportIDスライスをuuid.UUIDスライスに変換する。
func ExportIDsToUUIDs(ids []ExportID) []uuid.UUID {
	us := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		us[i] = uuid.UUID(id)
	}
	return us
}

// UUIDsToExportIDsはuuid.UUIDスライスをExportIDスライスに変換する。
func UUIDsToExportIDs(us []uuid.UUID) []ExportID {
	ids := make([]ExportID, len(us))
	for i, u := range us {
		ids[i] = ExportID(u)
	}
	return ids
}
