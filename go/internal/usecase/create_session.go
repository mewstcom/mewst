// Package usecase はビジネスロジックとトランザクション管理を担当します
package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// CreateSessionUsecase はセッション作成のユースケース
type CreateSessionUsecase struct {
	actorRepo   *repository.ActorRepository
	sessionRepo *repository.SessionRepository
}

// NewCreateSessionUsecase はCreateSessionUsecaseを生成する
func NewCreateSessionUsecase(actorRepo *repository.ActorRepository, sessionRepo *repository.SessionRepository) *CreateSessionUsecase {
	return &CreateSessionUsecase{
		actorRepo:   actorRepo,
		sessionRepo: sessionRepo,
	}
}

// CreateSessionInput はセッション作成の入力パラメータ
type CreateSessionInput struct {
	UserID    uuid.UUID
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
	// ユーザーIDからアクターを取得
	actor, err := uc.actorRepo.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("アクターの取得に失敗: %w", err)
	}

	// セッショントークンを生成
	token, err := auth.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("セッショントークンの生成に失敗: %w", err)
	}

	// セッションを作成
	s, err := uc.sessionRepo.Create(ctx, repository.CreateSessionParams{
		ActorID:   actor.ID,
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
