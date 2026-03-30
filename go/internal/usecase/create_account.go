package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// ProfileOwnerTypeUser はプロフィールの所有者タイプ（ユーザー）
const ProfileOwnerTypeUser = "User"

// DefaultAvatarKind はデフォルトのアバター種別
const DefaultAvatarKind = "default"

// CreateAccountUsecase はアカウント作成のユースケース
type CreateAccountUsecase struct {
	db              *sql.DB
	userRepo        *repository.UserRepository
	profileRepo     *repository.ProfileRepository
	userProfileRepo *repository.UserProfileRepository
	actorRepo       *repository.ActorRepository
}

// NewCreateAccountUsecase はCreateAccountUsecaseを生成する
func NewCreateAccountUsecase(
	db *sql.DB,
	userRepo *repository.UserRepository,
	profileRepo *repository.ProfileRepository,
	userProfileRepo *repository.UserProfileRepository,
	actorRepo *repository.ActorRepository,
) *CreateAccountUsecase {
	return &CreateAccountUsecase{
		db:              db,
		userRepo:        userRepo,
		profileRepo:     profileRepo,
		userProfileRepo: userProfileRepo,
		actorRepo:       actorRepo,
	}
}

// CreateAccountInput はアカウント作成の入力パラメータ
type CreateAccountInput struct {
	Email    string
	Atname   string
	Password string
	Locale   string
	TimeZone string
}

// CreateAccountResult はアカウント作成の結果
type CreateAccountResult struct {
	Actor *model.Actor
}

// Execute はアカウントを作成する
// Profile, User, UserProfile, Actor を一括で作成し、トランザクション管理を行う
func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountResult, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// トランザクション内で操作するためのリポジトリを取得
	userRepo := uc.userRepo.WithTx(tx)
	profileRepo := uc.profileRepo.WithTx(tx)
	userProfileRepo := uc.userProfileRepo.WithTx(tx)
	actorRepo := uc.actorRepo.WithTx(tx)

	currentTime := time.Now()

	// 1. Profile を作成
	profile, err := profileRepo.Create(ctx, repository.CreateProfileParams{
		OwnerType:     ProfileOwnerTypeUser,
		Atname:        input.Atname,
		Name:          "",
		Description:   "",
		ImageURL:      "",
		JoinedAt:      currentTime,
		AvatarKind:    DefaultAvatarKind,
		GravatarEmail: "",
		GravatarURL:   "",
	})
	if err != nil {
		return nil, fmt.Errorf("プロフィールの作成に失敗: %w", err)
	}

	// 2. パスワードをハッシュ化
	passwordDigest, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("パスワードのハッシュ化に失敗: %w", err)
	}

	// 3. User を作成
	user, err := userRepo.Create(ctx, repository.CreateUserParams{
		Email:          input.Email,
		PasswordDigest: passwordDigest,
		Locale:         input.Locale,
		TimeZone:       input.TimeZone,
	})
	if err != nil {
		return nil, fmt.Errorf("ユーザーの作成に失敗: %w", err)
	}

	// 4. UserProfile を作成
	_, err = userProfileRepo.Create(ctx, repository.CreateUserProfileParams{
		UserID:    user.ID,
		ProfileID: profile.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("ユーザープロフィール関連付けの作成に失敗: %w", err)
	}

	// 5. Actor を作成
	actor, err := actorRepo.Create(ctx, repository.CreateActorParams{
		UserID:    user.ID,
		ProfileID: profile.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("アクターの作成に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return &CreateAccountResult{
		Actor: actor,
	}, nil
}
