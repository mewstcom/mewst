package usecase_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// setupCreateAccountTest はテスト用のユースケースとリポジトリをセットアップする
func setupCreateAccountTest(t *testing.T) (
	*usecase.CreateAccountUsecase,
	*repository.UserRepository,
	*repository.ProfileRepository,
	*repository.UserProfileRepository,
	*repository.ActorRepository,
	context.Context,
) {
	t.Helper()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	userProfileRepo := repository.NewUserProfileRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)

	uc := usecase.NewCreateAccountUsecase(db, userRepo, profileRepo, userProfileRepo, actorRepo)

	return uc, userRepo, profileRepo, userProfileRepo, actorRepo, ctx
}

func TestCreateAccountUsecase_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input usecase.CreateAccountInput
	}{
		{
			name: "正常系: 基本的な入力でアカウントを作成できる",
			input: usecase.CreateAccountInput{
				Email:    "newuser@example.com",
				Atname:   "newuser",
				Password: "securePassword123",
				Locale:   "ja",
				TimeZone: "Asia/Tokyo",
			},
		},
		{
			name: "正常系: 英語ロケールでアカウントを作成できる",
			input: usecase.CreateAccountInput{
				Email:    "english@example.com",
				Atname:   "englishuser",
				Password: "testPassword123",
				Locale:   "en",
				TimeZone: "UTC",
			},
		},
		{
			name: "正常系: 日本語パスワードでアカウントを作成できる",
			input: usecase.CreateAccountInput{
				Email:    "japanese-pw-execute@example.com",
				Atname:   "japanesepwexec",
				Password: "パスワード安全123",
				Locale:   "ja",
				TimeZone: "Asia/Tokyo",
			},
		},
		{
			name: "正常系: 最大長のアットネーム（20文字）でアカウントを作成できる",
			input: usecase.CreateAccountInput{
				Email:    "long-atname@example.com",
				Atname:   "abcdefghij1234567890",
				Password: "testPassword123",
				Locale:   "ja",
				TimeZone: "Asia/Tokyo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, userRepo, profileRepo, userProfileRepo, actorRepo, ctx := setupCreateAccountTest(t)

			result, err := uc.Execute(ctx, tt.input)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if result == nil {
				t.Fatal("Execute() result should not be nil")
			}

			if result.Actor == nil {
				t.Fatal("Actor should not be nil")
			}

			// Actorの検証
			actor, err := actorRepo.GetByID(ctx, result.Actor.ID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if actor.UserID != result.Actor.UserID {
				t.Errorf("Actor.UserID = %v, want %v", actor.UserID, result.Actor.UserID)
			}

			if actor.ProfileID != result.Actor.ProfileID {
				t.Errorf("Actor.ProfileID = %v, want %v", actor.ProfileID, result.Actor.ProfileID)
			}

			// Userの検証
			user, err := userRepo.GetByID(ctx, result.Actor.UserID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if user.Email != tt.input.Email {
				t.Errorf("User.Email = %v, want %v", user.Email, tt.input.Email)
			}

			if user.Locale != tt.input.Locale {
				t.Errorf("User.Locale = %v, want %v", user.Locale, tt.input.Locale)
			}

			if user.TimeZone != tt.input.TimeZone {
				t.Errorf("User.TimeZone = %v, want %v", user.TimeZone, tt.input.TimeZone)
			}

			// Profileの検証
			profile, err := profileRepo.GetByID(ctx, result.Actor.ProfileID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if profile.Atname != tt.input.Atname {
				t.Errorf("Profile.Atname = %v, want %v", profile.Atname, tt.input.Atname)
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

			// UserProfileの検証
			userProfile, err := userProfileRepo.GetByUserID(ctx, result.Actor.UserID)
			if err != nil {
				t.Fatalf("GetByUserID() error = %v", err)
			}

			if userProfile.UserID != result.Actor.UserID {
				t.Errorf("UserProfile.UserID = %v, want %v", userProfile.UserID, result.Actor.UserID)
			}

			if userProfile.ProfileID != result.Actor.ProfileID {
				t.Errorf("UserProfile.ProfileID = %v, want %v", userProfile.ProfileID, result.Actor.ProfileID)
			}
		})
	}
}

func TestCreateAccountUsecase_Execute_HashesPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		atname   string
		password string
	}{
		{
			name:     "ASCII文字のパスワードがハッシュ化される",
			email:    "ascii-password@example.com",
			atname:   "asciipassword",
			password: "securePassword123",
		},
		{
			name:     "日本語を含むパスワードがハッシュ化される",
			email:    "japanese-password@example.com",
			atname:   "japanesepassword",
			password: "パスワード安全123",
		},
		{
			name:     "記号を含むパスワードがハッシュ化される",
			email:    "symbol-password@example.com",
			atname:   "symbolpassword",
			password: "P@ssw0rd!#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, userRepo, _, _, _, ctx := setupCreateAccountTest(t)

			result, err := uc.Execute(ctx, usecase.CreateAccountInput{
				Email:    tt.email,
				Atname:   tt.atname,
				Password: tt.password,
				Locale:   "ja",
				TimeZone: "Asia/Tokyo",
			})

			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			user, err := userRepo.GetByID(ctx, result.Actor.UserID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			// パスワードが平文で保存されていないことを確認
			if user.PasswordDigest == tt.password {
				t.Error("Password should be hashed, not stored as plain text")
			}

			// bcryptでハッシュ化されていることを確認
			err = auth.CheckPassword(user.PasswordDigest, tt.password)
			if err != nil {
				t.Errorf("Password should be verifiable with bcrypt: %v", err)
			}

			// 間違ったパスワードでは検証できないことを確認
			err = auth.CheckPassword(user.PasswordDigest, "wrongPassword")
			if err == nil {
				t.Error("Wrong password should not be verifiable")
			}
		})
	}
}
