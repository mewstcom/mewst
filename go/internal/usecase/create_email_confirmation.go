package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// CreateEmailConfirmationUsecase はメール確認作成のユースケース
type CreateEmailConfirmationUsecase struct {
	emailConfirmRepo *repository.EmailConfirmationRepository
	dispatcher       *dispatcher.Dispatcher
}

// NewCreateEmailConfirmationUsecase はCreateEmailConfirmationUsecaseを生成する
func NewCreateEmailConfirmationUsecase(
	emailConfirmRepo *repository.EmailConfirmationRepository,
	d *dispatcher.Dispatcher,
) *CreateEmailConfirmationUsecase {
	return &CreateEmailConfirmationUsecase{
		emailConfirmRepo: emailConfirmRepo,
		dispatcher:       d,
	}
}

// CreateEmailConfirmationInput はメール確認作成の入力パラメータ
type CreateEmailConfirmationInput struct {
	Email  string
	Event  model.EmailConfirmationEvent
	Locale string
}

// CreateEmailConfirmationResult はメール確認作成の結果
type CreateEmailConfirmationResult struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute はメール確認を作成し、確認メール送信ジョブをエンキューする
func (uc *CreateEmailConfirmationUsecase) Execute(ctx context.Context, input CreateEmailConfirmationInput) (*CreateEmailConfirmationResult, error) {
	// 6桁のランダムなコードを生成
	code, err := generateConfirmationCode()
	if err != nil {
		return nil, fmt.Errorf("確認コードの生成に失敗: %w", err)
	}

	// メール確認レコードを作成
	ec, err := uc.emailConfirmRepo.Create(ctx, repository.CreateEmailConfirmationParams{
		Email: input.Email,
		Event: input.Event,
		Code:  code,
	})
	if err != nil {
		return nil, fmt.Errorf("メール確認レコードの作成に失敗: %w", err)
	}

	// メール送信ジョブをエンキュー（テンプレートのレンダリングはWorkerで行う）
	err = uc.dispatcher.EnqueueEmailConfirmation(ctx, dispatcher.SendEmailConfirmationArgs{
		Email:  input.Email,
		Code:   code,
		Locale: input.Locale,
	})
	if err != nil {
		// ジョブエンキューに失敗してもコードは有効なので、エラーログを出力して続行
		slog.ErrorContext(ctx, "メール送信ジョブのエンキューに失敗しました",
			"email", input.Email,
			"error", err,
		)
	} else {
		slog.InfoContext(ctx, "メール送信ジョブをエンキューしました",
			"email", input.Email,
		)
	}

	return &CreateEmailConfirmationResult{
		EmailConfirmation: ec,
	}, nil
}

// generateConfirmationCode は6桁のランダムな数字コードを生成する
func generateConfirmationCode() (string, error) {
	// 000000 から 999999 までのランダムな数を生成
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	// 6桁になるようにゼロ埋め
	return fmt.Sprintf("%06d", n.Int64()), nil
}
