package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// CreateSignUpUsecase はサインアップのユースケース
type CreateSignUpUsecase struct {
	signUpValidator  *validator.SignUpCreateValidator
	emailConfirmRepo *repository.EmailConfirmationRepository
	dispatcher       *dispatcher.Dispatcher
}

// NewCreateSignUpUsecase は CreateSignUpUsecase を生成する
func NewCreateSignUpUsecase(
	signUpValidator *validator.SignUpCreateValidator,
	emailConfirmRepo *repository.EmailConfirmationRepository,
	d *dispatcher.Dispatcher,
) *CreateSignUpUsecase {
	return &CreateSignUpUsecase{
		signUpValidator:  signUpValidator,
		emailConfirmRepo: emailConfirmRepo,
		dispatcher:       d,
	}
}

// CreateSignUpInput はサインアップの入力パラメータ
type CreateSignUpInput struct {
	Email  string
	Locale string
}

// CreateSignUpOutput はサインアップの出力パラメータ
type CreateSignUpOutput struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute はサインアップ処理を実行する
func (uc *CreateSignUpUsecase) Execute(ctx context.Context, input CreateSignUpInput) (*CreateSignUpOutput, error) {
	// 1. バリデーション (トランザクション外)
	if err := uc.signUpValidator.Validate(ctx, validator.SignUpCreateValidatorInput{
		Email: input.Email,
	}); err != nil {
		return nil, err
	}

	// 2. ビジネスロジック + 永続化
	return uc.createSignUp(ctx, input)
}

// createSignUp は確認コードを生成し、メール確認レコードの作成とメール送信ジョブのエンキューを行う
func (uc *CreateSignUpUsecase) createSignUp(ctx context.Context, input CreateSignUpInput) (*CreateSignUpOutput, error) {
	code, err := generateConfirmationCode()
	if err != nil {
		return nil, fmt.Errorf("確認コードの生成に失敗: %w", err)
	}

	ec, err := uc.emailConfirmRepo.Create(ctx, repository.CreateEmailConfirmationInput{
		Email: input.Email,
		Event: model.EmailConfirmationEventSignUp,
		Code:  code,
	})
	if err != nil {
		return nil, fmt.Errorf("メール確認レコードの作成に失敗: %w", err)
	}

	// メール送信ジョブをエンキュー。
	// 失敗時はログのみ残して正常完了する: 確認レコードは作成済みのためユーザーは再申請で回復可能であり、
	// ジョブキュー障害で 500 を返すのは過剰な扱いになるため。
	if err := uc.dispatcher.EnqueueEmailConfirmation(ctx, input.Email, code, input.Locale); err != nil {
		slog.ErrorContext(ctx, "メール送信ジョブのエンキューに失敗しました",
			"email", input.Email,
			"error", err,
		)
	} else {
		slog.InfoContext(ctx, "メール送信ジョブをエンキューしました",
			"email", input.Email,
		)
	}

	return &CreateSignUpOutput{
		EmailConfirmation: ec,
	}, nil
}
