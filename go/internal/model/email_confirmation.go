package model

import (
	"time"
)

// EmailConfirmationEvent はメール確認のイベント種別
type EmailConfirmationEvent string

const (
	// EmailConfirmationEventPasswordReset はパスワードリセットイベント
	EmailConfirmationEventPasswordReset EmailConfirmationEvent = "password_reset"
	// EmailConfirmationEventSignUp はサインアップイベント
	EmailConfirmationEventSignUp EmailConfirmationEvent = "sign_up"
	// EmailConfirmationEventEmailUpdate はメールアドレス更新イベント
	EmailConfirmationEventEmailUpdate EmailConfirmationEvent = "email_update"
)

// EmailConfirmation はメール確認のドメインモデル
type EmailConfirmation struct {
	ID          EmailConfirmationID
	Email       string
	Event       EmailConfirmationEvent
	Code        string
	SucceededAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EmailConfirmationExpirationMinutes は確認コードの有効期限（分）
const EmailConfirmationExpirationMinutes = 15

// IsExpired は確認コードが有効期限切れかどうかを返す
func (ec *EmailConfirmation) IsExpired() bool {
	expirationTime := ec.CreatedAt.Add(time.Duration(EmailConfirmationExpirationMinutes) * time.Minute)
	return time.Now().After(expirationTime)
}

// IsSucceeded は確認が成功済みかどうかを返す
func (ec *EmailConfirmation) IsSucceeded() bool {
	return ec.SucceededAt != nil
}
