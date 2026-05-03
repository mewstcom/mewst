package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// CreateSignInUsecase はサインインユースケース
type CreateSignInUsecase struct {
	signInValidator *validator.SignInCreateValidator
	actorRepo       *repository.ActorRepository
	sessionRepo     *repository.SessionRepository
}

// NewCreateSignInUsecase は CreateSignInUsecase を生成する
func NewCreateSignInUsecase(
	signInValidator *validator.SignInCreateValidator,
	actorRepo *repository.ActorRepository,
	sessionRepo *repository.SessionRepository,
) *CreateSignInUsecase {
	return &CreateSignInUsecase{
		signInValidator: signInValidator,
		actorRepo:       actorRepo,
		sessionRepo:     sessionRepo,
	}
}

// CreateSignInInput はサインインの入力パラメータ
type CreateSignInInput struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

// CreateSignInOutput はサインインの出力パラメータ
type CreateSignInOutput struct {
	Session *model.Session
	Token   string
}

// Execute はサインイン処理を実行する
func (uc *CreateSignInUsecase) Execute(ctx context.Context, input CreateSignInInput) (*CreateSignInOutput, error) {
	// 1. バリデーション (トランザクション外)
	user, err := uc.signInValidator.Validate(ctx, validator.SignInCreateValidatorInput{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return nil, err
	}

	// 2. データ取得: Validator が引いたユーザーを使ってアクターを取得 (トランザクション外)
	actor, err := uc.actorRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("アクターの取得に失敗: %w", err)
	}
	if actor == nil {
		return nil, ErrNotFound
	}

	// 3. ビジネスロジック + 永続化
	return uc.createSignIn(ctx, actor.ID, input)
}

// createSignIn はトークンを生成しセッションを作成する
func (uc *CreateSignInUsecase) createSignIn(ctx context.Context, actorID model.ActorID, input CreateSignInInput) (*CreateSignInOutput, error) {
	token, err := auth.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("セッショントークンの生成に失敗: %w", err)
	}

	s, err := uc.sessionRepo.Create(ctx, repository.CreateSessionInput{
		ActorID:   actorID,
		Token:     token,
		IPAddress: input.IPAddress,
		UserAgent: input.UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("セッションの作成に失敗: %w", err)
	}

	return &CreateSignInOutput{
		Session: s,
		Token:   token,
	}, nil
}
