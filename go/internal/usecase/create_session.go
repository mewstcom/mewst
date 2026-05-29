// Package usecase はビジネスロジックとトランザクション管理を担当します
package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/i18n"
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
	UserID    model.UserID
	IPAddress string
	UserAgent string
}

// CreateSessionOutput はセッション作成の出力パラメータ
type CreateSessionOutput struct {
	Session *model.Session
	Token   string
}

// Execute はセッションを作成する
func (uc *CreateSessionUsecase) Execute(ctx context.Context, input CreateSessionInput) (*CreateSessionOutput, error) {
	// 1. データ取得 (トランザクション外)
	actor, err := uc.actorRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("アクターの取得に失敗: %w", err)
	}
	if actor == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// 2. ビジネスロジック + 永続化
	return uc.createSession(ctx, actor.ID, input)
}

// createSession はトークンを生成しセッションを作成する
func (uc *CreateSessionUsecase) createSession(ctx context.Context, actorID model.ActorID, input CreateSessionInput) (*CreateSessionOutput, error) {
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

	return &CreateSessionOutput{
		Session: s,
		Token:   token,
	}, nil
}
