package testutil_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// TestNewProfileOwner pins the three relations the helper promises: the profile
// is owned by a user, the user_profiles row records that ownership, and the
// returned actor acts as that same user on that same profile. The expected
// values are literals rather than the constants the helper uses, so changing a
// constant does not move the test along with it.
//
// [Ja] TestNewProfileOwner はヘルパーが約束する 3 つの関係を固定する。プロフィールが
// ユーザーに所有されていること、user_profiles 行がその所有関係を記録していること、
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
		t.Fatalf("querying the profile ownership fixture: %v", err)
	}

	if ownerType != "User" {
		t.Errorf("owner type = %q, want %q", ownerType, "User")
	}
	if got := model.UserID(userID); got != owner.UserID {
		t.Errorf("owner user ID = %v, want %v", got, owner.UserID)
	}
	if got := model.UserID(actorUserID); got != owner.UserID {
		t.Errorf("actor user ID = %v, want %v", got, owner.UserID)
	}
	if got := model.ProfileID(actorProfileID); got != owner.ProfileID {
		t.Errorf("actor profile ID = %v, want %v", got, owner.ProfileID)
	}
}
