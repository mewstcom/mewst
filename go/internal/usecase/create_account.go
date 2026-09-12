package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// DefaultAvatarKind はデフォルトのアバター種別
const DefaultAvatarKind = "default"

// CreateAccountUsecase はアカウント作成のユースケース
type CreateAccountUsecase struct {
	db               *sql.DB
	accountValidator *validator.AccountCreateValidator
	userRepo         *repository.UserRepository
	profileRepo      *repository.ProfileRepository
	userProfileRepo  *repository.UserProfileRepository
	actorRepo        *repository.ActorRepository
}

// NewCreateAccountUsecase はCreateAccountUsecaseを生成する
func NewCreateAccountUsecase(
	db *sql.DB,
	accountValidator *validator.AccountCreateValidator,
	userRepo *repository.UserRepository,
	profileRepo *repository.ProfileRepository,
	userProfileRepo *repository.UserProfileRepository,
	actorRepo *repository.ActorRepository,
) *CreateAccountUsecase {
	return &CreateAccountUsecase{
		db:               db,
		accountValidator: accountValidator,
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		userProfileRepo:  userProfileRepo,
		actorRepo:        actorRepo,
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

// CreateAccountOutput はアカウント作成の出力パラメータ
type CreateAccountOutput struct {
	Actor *model.Actor
}

// Execute はアカウントを作成する
// Profile, User, UserProfile, Actor を一括で作成し、トランザクション管理を行う
func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error) {
	// 1. バリデーション (トランザクション外)
	if err := uc.accountValidator.Validate(ctx, validator.AccountCreateValidatorInput{
		Email:    input.Email,
		Atname:   input.Atname,
		Password: input.Password,
	}); err != nil {
		return nil, err
	}

	// 2. CPU 計算 (bcrypt) と時刻取得をトランザクション外で済ませる。
	// bcrypt はコスト 10 で 100ms 級の処理になるため、トランザクション内で実行すると
	// その間 DB 接続を専有してロック競合の原因になる。
	passwordDigest, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("パスワードのハッシュ化に失敗: %w", err)
	}
	currentTime := time.Now()

	// 3. ビジネスロジック + 永続化
	return uc.createAccount(ctx, input, passwordDigest, currentTime)
}

// createAccount は Profile / User / UserProfile / Actor を 1 トランザクションで作成する
func (uc *CreateAccountUsecase) createAccount(ctx context.Context, input CreateAccountInput, passwordDigest string, currentTime time.Time) (*CreateAccountOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	userRepo := uc.userRepo.WithTx(tx)
	profileRepo := uc.profileRepo.WithTx(tx)
	userProfileRepo := uc.userProfileRepo.WithTx(tx)
	actorRepo := uc.actorRepo.WithTx(tx)

	profile, err := profileRepo.Create(ctx, repository.CreateProfileInput{
		OwnerType:     model.ProfileOwnerTypeUser,
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

	user, err := userRepo.Create(ctx, repository.CreateUserInput{
		Email:          input.Email,
		PasswordDigest: passwordDigest,
		Locale:         input.Locale,
		TimeZone:       input.TimeZone,
	})
	if err != nil {
		return nil, fmt.Errorf("ユーザーの作成に失敗: %w", err)
	}

	if _, err := userProfileRepo.Create(ctx, repository.CreateUserProfileInput{
		UserID:    user.ID,
		ProfileID: profile.ID,
	}); err != nil {
		return nil, fmt.Errorf("ユーザープロフィール関連付けの作成に失敗: %w", err)
	}

	actor, err := actorRepo.Create(ctx, repository.CreateActorInput{
		UserID:    user.ID,
		ProfileID: profile.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("アクターの作成に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return &CreateAccountOutput{
		Actor: actor,
	}, nil
}
