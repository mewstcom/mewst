package model

import (
	"time"
)

// EmailConfirmationEventはメール確認のイベント種別
type EmailConfirmationEvent string

const (
	// EmailConfirmationEventPasswordResetはパスワードリセットイベント
	EmailConfirmationEventPasswordReset EmailConfirmationEvent = "password_reset"
	// EmailConfirmationEventSignUpはサインアップイベント
	EmailConfirmationEventSignUp EmailConfirmationEvent = "sign_up"
	// EmailConfirmationEventEmailUpdateはメールアドレス更新イベント
	EmailConfirmationEventEmailUpdate EmailConfirmationEvent = "email_update"
)

// EmailConfirmationはメール確認のドメインモデル
type EmailConfirmation struct {
	ID          EmailConfirmationID
	Email       string
	Event       EmailConfirmationEvent
	Code        string
	SucceededAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EmailConfirmationExpirationMinutesは確認コードの有効期限 (分)
const EmailConfirmationExpirationMinutes = 15

// IsExpiredは確認コードが有効期限切れかどうかを返す
func (ec *EmailConfirmation) IsExpired() bool {
	expirationTime := ec.CreatedAt.Add(time.Duration(EmailConfirmationExpirationMinutes) * time.Minute)
	return time.Now().After(expirationTime)
}

// IsSucceededは確認が成功済みかどうかを返す
func (ec *EmailConfirmation) IsSucceeded() bool {
	return ec.SucceededAt != nil
}
