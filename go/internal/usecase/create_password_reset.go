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

// CreatePasswordResetUsecase はパスワードリセット開始のユースケース
type CreatePasswordResetUsecase struct {
	passwordResetValidator *validator.PasswordResetCreateValidator
	emailConfirmRepo       *repository.EmailConfirmationRepository
	dispatcher             *dispatcher.Dispatcher
}

// NewCreatePasswordResetUsecase は CreatePasswordResetUsecase を生成する
func NewCreatePasswordResetUsecase(
	passwordResetValidator *validator.PasswordResetCreateValidator,
	emailConfirmRepo *repository.EmailConfirmationRepository,
	d *dispatcher.Dispatcher,
) *CreatePasswordResetUsecase {
	return &CreatePasswordResetUsecase{
		passwordResetValidator: passwordResetValidator,
		emailConfirmRepo:       emailConfirmRepo,
		dispatcher:             d,
	}
}

// CreatePasswordResetInput はパスワードリセットの入力パラメータ
type CreatePasswordResetInput struct {
	Email  string
	Locale string
}

// CreatePasswordResetOutput はパスワードリセットの出力パラメータ
type CreatePasswordResetOutput struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute はパスワードリセット処理を実行する
func (uc *CreatePasswordResetUsecase) Execute(ctx context.Context, input CreatePasswordResetInput) (*CreatePasswordResetOutput, error) {
	// 1. バリデーション (トランザクション外)
	if err := uc.passwordResetValidator.Validate(ctx, validator.PasswordResetCreateValidatorInput{
		Email: input.Email,
	}); err != nil {
		return nil, err
	}

	// 2. ビジネスロジック + 永続化
	return uc.createPasswordReset(ctx, input)
}

// createPasswordReset は確認コードを生成し、メール確認レコードの作成とメール送信ジョブのエンキューを行う
func (uc *CreatePasswordResetUsecase) createPasswordReset(ctx context.Context, input CreatePasswordResetInput) (*CreatePasswordResetOutput, error) {
	code, err := generateConfirmationCode()
	if err != nil {
		return nil, fmt.Errorf("確認コードの生成に失敗: %w", err)
	}

	ec, err := uc.emailConfirmRepo.Create(ctx, repository.CreateEmailConfirmationInput{
		Email: input.Email,
		Event: model.EmailConfirmationEventPasswordReset,
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

	return &CreatePasswordResetOutput{
		EmailConfirmation: ec,
	}, nil
}
