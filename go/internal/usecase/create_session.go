// Package usecase はビジネスロジックとトランザクション管理を担当します
package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
)

// CreateSessionUsecase はセッション作成のユースケース
type CreateSessionUsecase struct {
	sessionRepo *repository.SessionRepository
}

// NewCreateSessionUsecase はCreateSessionUsecaseを生成する
func NewCreateSessionUsecase(sessionRepo *repository.SessionRepository) *CreateSessionUsecase {
	return &CreateSessionUsecase{
		sessionRepo: sessionRepo,
	}
}

// CreateSessionInput はセッション作成の入力パラメータ
type CreateSessionInput struct {
	ActorID   uuid.UUID
	IPAddress string
	UserAgent string
}

// CreateSessionResult はセッション作成の結果
type CreateSessionResult struct {
	Session *model.Session
	Token   string
}

// Execute はセッションを作成する
func (uc *CreateSessionUsecase) Execute(ctx context.Context, input CreateSessionInput) (*CreateSessionResult, error) {
	// セッショントークンを生成
	token, err := session.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("セッショントークンの生成に失敗: %w", err)
	}

	// セッションを作成
	s, err := uc.sessionRepo.Create(ctx, repository.CreateSessionParams{
		ActorID:   input.ActorID,
		Token:     token,
		IPAddress: input.IPAddress,
		UserAgent: input.UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("セッションの作成に失敗: %w", err)
	}

	return &CreateSessionResult{
		Session: s,
		Token:   token,
	}, nil
}
