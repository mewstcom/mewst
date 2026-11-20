package testutil_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// TestNewProfileOwnerはヘルパーが約束する3つの関係を固定する。プロフィールが
// ユーザーに所有されていること、user_profiles行がその所有関係を記録していること、
// 返されるアクターが同じユーザーとして同じプロフィール上で活動することである。
// 期待値はヘルパーが使う定数ではなくリテラルにする。定数を変えたときにテストも
// 一緒に動いてしまわないようにするため。
func TestNewProfileOwner(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)

	var ownerType string
	var userID, actorUserID, actorProfileID uuid.UUID
	err := tx.QueryRow(`
		SELECT profiles.owner_type, user_profiles.user_id, actors.user_id, actors.profile_id
		FROM profiles
		JOIN user_profiles ON user_profiles.profile_id = profiles.id
		JOIN actors ON actors.id = $2
		WHERE profiles.id = $1
	`, uuid.UUID(owner.ProfileID), uuid.UUID(owner.ActorID)).Scan(&ownerType, &userID, &actorUserID, &actorProfileID)
	if err != nil {
		t.Fatalf("プロフィールの所有関係のフィクスチャの取得に失敗: %v", err)
	}

	if ownerType != "User" {
		t.Errorf("所有者の種類 = %q、期待値 = %q", ownerType, "User")
	}
	if got := model.UserID(userID); got != owner.UserID {
		t.Errorf("所有者のユーザーID = %v、期待値 = %v", got, owner.UserID)
	}
	if got := model.UserID(actorUserID); got != owner.UserID {
		t.Errorf("actorのユーザーID = %v、期待値 = %v", got, owner.UserID)
	}
	if got := model.ProfileID(actorProfileID); got != owner.ProfileID {
		t.Errorf("actorのプロフィールID = %v、期待値 = %v", got, owner.ProfileID)
	}
}
