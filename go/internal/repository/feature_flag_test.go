package repository_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestFeatureFlagRepository_IsEnabledForActor(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストデータを作成 (User → Profile → Actor)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("ff-actor@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("ffactoruser").
		Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	repo := repository.NewFeatureFlagRepository(testutil.QueriesWithTx(tx))

	t.Run("アクターに対してフラグが有効な場合 true を返す", func(t *testing.T) {
		_ = testutil.NewFeatureFlagBuilder(t, tx).
			WithActorID(actorID).
			WithName(model.FeatureFlagExample).
			Build()

		enabled, err := repo.IsEnabledForActor(ctx, actorID, model.FeatureFlagExample)
		if err != nil {
			t.Fatalf("IsEnabledForActor() error = %v", err)
		}
		if !enabled {
			t.Error("IsEnabledForActor() = false, want true")
		}
	})

	t.Run("フラグが存在しないアクターでは false を返す", func(t *testing.T) {
		// 別のアクターを作成 (フラグは紐付けない)
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("ff-actor-other@example.com").
			Build()
		otherProfileID := testutil.NewProfileBuilder(t, tx).
			WithAtname("ffactorother").
			Build()
		otherActorID := testutil.NewActorBuilder(t, tx).
			WithUserID(otherUserID).
			WithProfileID(otherProfileID).
			Build()

		enabled, err := repo.IsEnabledForActor(ctx, otherActorID, model.FeatureFlagExample)
		if err != nil {
			t.Fatalf("IsEnabledForActor() error = %v", err)
		}
		if enabled {
			t.Error("IsEnabledForActor() = true, want false")
		}
	})

	t.Run("別の名前のフラグでは false を返す", func(t *testing.T) {
		enabled, err := repo.IsEnabledForActor(ctx, actorID, model.FeatureFlagName("go_unknown"))
		if err != nil {
			t.Fatalf("IsEnabledForActor() error = %v", err)
		}
		if enabled {
			t.Error("IsEnabledForActor() = true, want false")
		}
	})
}

func TestFeatureFlagRepository_IsEnabledForDevice(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	repo := repository.NewFeatureFlagRepository(testutil.QueriesWithTx(tx))

	t.Run("device_token が一致する場合 true を返す", func(t *testing.T) {
		deviceToken := "device-token-match"
		_ = testutil.NewFeatureFlagBuilder(t, tx).
			WithDeviceToken(deviceToken).
			WithName(model.FeatureFlagExample).
			Build()

		enabled, err := repo.IsEnabledForDevice(ctx, deviceToken, "", model.FeatureFlagExample)
		if err != nil {
			t.Fatalf("IsEnabledForDevice() error = %v", err)
		}
		if !enabled {
			t.Error("IsEnabledForDevice() = false, want true")
		}
	})

	t.Run("セッショントークン経由で actor が一致する場合 true を返す", func(t *testing.T) {
		// User → Profile → Actor → Session を作成し、actor にフラグを紐付ける
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("ff-device-actor@example.com").
			Build()
		profileID := testutil.NewProfileBuilder(t, tx).
			WithAtname("ffdeviceactor").
			Build()
		actorID := testutil.NewActorBuilder(t, tx).
			WithUserID(userID).
			WithProfileID(profileID).
			Build()
		sessionToken := "session-token-for-actor-flag"
		_ = testutil.NewSessionBuilder(t, tx).
			WithActorID(actorID).
			WithToken(sessionToken).
			Build()
		_ = testutil.NewFeatureFlagBuilder(t, tx).
			WithActorID(actorID).
			WithName(model.FeatureFlagExample).
			Build()

		enabled, err := repo.IsEnabledForDevice(ctx, "", sessionToken, model.FeatureFlagExample)
		if err != nil {
			t.Fatalf("IsEnabledForDevice() error = %v", err)
		}
		if !enabled {
			t.Error("IsEnabledForDevice() = false, want true")
		}
	})

	t.Run("device_token もセッションも一致しない場合 false を返す", func(t *testing.T) {
		_ = testutil.NewFeatureFlagBuilder(t, tx).
			WithDeviceToken("device-token-existing").
			WithName(model.FeatureFlagName("go_no_match")).
			Build()

		enabled, err := repo.IsEnabledForDevice(ctx, "device-token-different", "unknown-session-token", model.FeatureFlagName("go_no_match"))
		if err != nil {
			t.Fatalf("IsEnabledForDevice() error = %v", err)
		}
		if enabled {
			t.Error("IsEnabledForDevice() = true, want false")
		}
	})

	t.Run("device_token と sessionToken の両方が空の場合 false を返す", func(t *testing.T) {
		// フラグが存在していても、識別子が無い閲覧者には有効と判定しない
		_ = testutil.NewFeatureFlagBuilder(t, tx).
			WithDeviceToken("device-token-some").
			WithName(model.FeatureFlagName("go_empty_identifier")).
			Build()

		enabled, err := repo.IsEnabledForDevice(ctx, "", "", model.FeatureFlagName("go_empty_identifier"))
		if err != nil {
			t.Fatalf("IsEnabledForDevice() error = %v", err)
		}
		if enabled {
			t.Error("IsEnabledForDevice() = true, want false")
		}
	})
}
