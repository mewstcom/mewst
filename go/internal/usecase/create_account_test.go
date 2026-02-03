package usecase_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestCreateAccountUsecase_Execute(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// リポジトリを作成
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	userProfileRepo := repository.NewUserProfileRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)

	// ユースケースを実行
	uc := usecase.NewCreateAccountUsecase(db, userRepo, profileRepo, userProfileRepo, actorRepo)
	result, err := uc.Execute(ctx, usecase.CreateAccountInput{
		Email:    "newuser@example.com",
		Atname:   "newuser",
		Password: "securePassword123",
		Locale:   "ja",
		TimeZone: "Asia/Tokyo",
	})

	// アサーション
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result == nil {
		t.Fatal("Execute() result should not be nil")
	}

	if result.Actor == nil {
		t.Fatal("Actor should not be nil")
	}

	// 作成されたActorを確認
	actor, err := actorRepo.GetByID(ctx, result.Actor.ID)
	if err != nil {
		t.Fatalf("GetActorByID() error = %v", err)
	}

	if actor.UserID != result.Actor.UserID {
		t.Errorf("Actor.UserID = %v, want %v", actor.UserID, result.Actor.UserID)
	}

	if actor.ProfileID != result.Actor.ProfileID {
		t.Errorf("Actor.ProfileID = %v, want %v", actor.ProfileID, result.Actor.ProfileID)
	}
}

func TestCreateAccountUsecase_Execute_CreatesUser(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	userProfileRepo := repository.NewUserProfileRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)

	uc := usecase.NewCreateAccountUsecase(db, userRepo, profileRepo, userProfileRepo, actorRepo)
	result, err := uc.Execute(ctx, usecase.CreateAccountInput{
		Email:    "user-creation-test@example.com",
		Atname:   "usercreate",
		Password: "testPassword123",
		Locale:   "en",
		TimeZone: "UTC",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 作成されたUserを確認
	user, err := userRepo.GetByID(ctx, result.Actor.UserID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	if user.Email != "user-creation-test@example.com" {
		t.Errorf("User.Email = %v, want %v", user.Email, "user-creation-test@example.com")
	}

	if user.Locale != "en" {
		t.Errorf("User.Locale = %v, want %v", user.Locale, "en")
	}

	if user.TimeZone != "UTC" {
		t.Errorf("User.TimeZone = %v, want %v", user.TimeZone, "UTC")
	}
}

func TestCreateAccountUsecase_Execute_CreatesProfile(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	userProfileRepo := repository.NewUserProfileRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)

	uc := usecase.NewCreateAccountUsecase(db, userRepo, profileRepo, userProfileRepo, actorRepo)
	result, err := uc.Execute(ctx, usecase.CreateAccountInput{
		Email:    "profile-test@example.com",
		Atname:   "profiletest",
		Password: "testPassword123",
		Locale:   "ja",
		TimeZone: "Asia/Tokyo",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 作成されたProfileを確認
	profile, err := profileRepo.GetByID(ctx, result.Actor.ProfileID)
	if err != nil {
		t.Fatalf("GetProfileByID() error = %v", err)
	}

	if profile.Atname != "profiletest" {
		t.Errorf("Profile.Atname = %v, want %v", profile.Atname, "profiletest")
	}

	if profile.OwnerType != usecase.ProfileOwnerTypeUser {
		t.Errorf("Profile.OwnerType = %v, want %v", profile.OwnerType, usecase.ProfileOwnerTypeUser)
	}

	if profile.AvatarKind != usecase.DefaultAvatarKind {
		t.Errorf("Profile.AvatarKind = %v, want %v", profile.AvatarKind, usecase.DefaultAvatarKind)
	}

	if profile.JoinedAt.IsZero() {
		t.Error("Profile.JoinedAt should not be zero")
	}
}

func TestCreateAccountUsecase_Execute_CreatesUserProfile(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	userProfileRepo := repository.NewUserProfileRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)

	uc := usecase.NewCreateAccountUsecase(db, userRepo, profileRepo, userProfileRepo, actorRepo)
	result, err := uc.Execute(ctx, usecase.CreateAccountInput{
		Email:    "userprofile-test@example.com",
		Atname:   "userprofiletest",
		Password: "testPassword123",
		Locale:   "ja",
		TimeZone: "Asia/Tokyo",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 作成されたUserProfileを確認
	userProfile, err := userProfileRepo.GetByUserID(ctx, result.Actor.UserID)
	if err != nil {
		t.Fatalf("GetUserProfileByUserID() error = %v", err)
	}

	if userProfile.UserID != result.Actor.UserID {
		t.Errorf("UserProfile.UserID = %v, want %v", userProfile.UserID, result.Actor.UserID)
	}

	if userProfile.ProfileID != result.Actor.ProfileID {
		t.Errorf("UserProfile.ProfileID = %v, want %v", userProfile.ProfileID, result.Actor.ProfileID)
	}
}

func TestCreateAccountUsecase_Execute_HashesPassword(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	userProfileRepo := repository.NewUserProfileRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)

	plainPassword := "securePassword123"

	uc := usecase.NewCreateAccountUsecase(db, userRepo, profileRepo, userProfileRepo, actorRepo)
	result, err := uc.Execute(ctx, usecase.CreateAccountInput{
		Email:    "password-test@example.com",
		Atname:   "passwordtest",
		Password: plainPassword,
		Locale:   "ja",
		TimeZone: "Asia/Tokyo",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 作成されたUserのパスワードがハッシュ化されていることを確認
	user, err := userRepo.GetByID(ctx, result.Actor.UserID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	// パスワードが平文で保存されていないことを確認
	if user.PasswordDigest == plainPassword {
		t.Error("Password should be hashed, not stored as plain text")
	}

	// bcryptでハッシュ化されていることを確認
	err = auth.CheckPassword(user.PasswordDigest, plainPassword)
	if err != nil {
		t.Errorf("Password should be verifiable with bcrypt: %v", err)
	}

	// 間違ったパスワードでは検証できないことを確認
	err = auth.CheckPassword(user.PasswordDigest, "wrongPassword")
	if err == nil {
		t.Error("Wrong password should not be verifiable")
	}
}

func TestCreateAccountUsecase_Execute_JapanesePassword(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	userProfileRepo := repository.NewUserProfileRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)

	// 日本語を含むパスワード
	japanesePassword := "パスワード安全123"

	uc := usecase.NewCreateAccountUsecase(db, userRepo, profileRepo, userProfileRepo, actorRepo)
	result, err := uc.Execute(ctx, usecase.CreateAccountInput{
		Email:    "japanese-password@example.com",
		Atname:   "japanesepassword",
		Password: japanesePassword,
		Locale:   "ja",
		TimeZone: "Asia/Tokyo",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 日本語パスワードが正しく保存され、検証できることを確認
	user, err := userRepo.GetByID(ctx, result.Actor.UserID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	err = auth.CheckPassword(user.PasswordDigest, japanesePassword)
	if err != nil {
		t.Errorf("Japanese password should be verifiable: %v", err)
	}
}

func TestCreateAccountUsecase_Execute_LongAtname(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	userProfileRepo := repository.NewUserProfileRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)

	// 20文字のアットネーム（最大長）
	longAtname := "abcdefghij1234567890"

	uc := usecase.NewCreateAccountUsecase(db, userRepo, profileRepo, userProfileRepo, actorRepo)
	result, err := uc.Execute(ctx, usecase.CreateAccountInput{
		Email:    "long-atname@example.com",
		Atname:   longAtname,
		Password: "testPassword123",
		Locale:   "ja",
		TimeZone: "Asia/Tokyo",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 長いアットネームが正しく保存されていることを確認
	profile, err := profileRepo.GetByID(ctx, result.Actor.ProfileID)
	if err != nil {
		t.Fatalf("GetProfileByID() error = %v", err)
	}

	if profile.Atname != longAtname {
		t.Errorf("Profile.Atname = %v, want %v", profile.Atname, longAtname)
	}
}
