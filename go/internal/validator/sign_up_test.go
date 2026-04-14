package validator

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestSignUpCreateValidator_EmptyEmail(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewSignUpCreateValidator(userRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	_, err := validator.Validate(ctx, SignUpCreateValidatorInput{
		Email: "",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("バリデーションエラーが発生するべきです")
	}

	emailErrors := ve.GetFieldErrors("email")
	if len(emailErrors) == 0 {
		t.Error("emailフィールドのエラーが発生するべきです")
	}
}

func TestSignUpCreateValidator_InvalidEmailFormat(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewSignUpCreateValidator(userRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	invalidEmails := []string{
		"invalid",
		"invalid@",
		"@example.com",
		"invalid@.com",
	}

	for _, email := range invalidEmails {
		t.Run(email, func(t *testing.T) {
			_, err := validator.Validate(ctx, SignUpCreateValidatorInput{
				Email: email,
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("バリデーションエラーが発生するべきです")
			}

			emailErrors := ve.GetFieldErrors("email")
			if len(emailErrors) == 0 {
				t.Error("emailフィールドのエラーが発生するべきです")
			}
		})
	}
}

func TestSignUpCreateValidator_ValidEmail(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewSignUpCreateValidator(userRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	validEmails := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user@subdomain.example.com",
	}

	for _, email := range validEmails {
		t.Run(email, func(t *testing.T) {
			_, err := validator.Validate(ctx, SignUpCreateValidatorInput{
				Email: email,
			})

			if err != nil {
				t.Errorf("バリデーションエラーが発生すべきではありません: %v", err)
			}
		})
	}
}

func TestSignUpCreateValidator_EmailAlreadyTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewSignUpCreateValidator(userRepo)

	// 既存のユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("existing@example.com").
		WithPasswordDigest("$2a$10$test").
		Build()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	_, err := validator.Validate(ctx, SignUpCreateValidatorInput{
		Email: "existing@example.com",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("バリデーションエラーが発生するべきです")
	}

	emailErrors := ve.GetFieldErrors("email")
	if len(emailErrors) == 0 {
		t.Error("emailフィールドのエラーが発生するべきです")
	}
}

func TestSignUpCreateValidator_EmailNotTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewSignUpCreateValidator(userRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	_, err := validator.Validate(ctx, SignUpCreateValidatorInput{
		Email: "signup-validator-not-taken@example.com",
	})

	if err != nil {
		t.Errorf("バリデーションエラーが発生すべきではありません: %v", err)
	}
}

func TestSignUpCreateValidator_CaseInsensitiveEmail(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewSignUpCreateValidator(userRepo)

	// 小文字のメールアドレスで既存ユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("existing@example.com").
		WithPasswordDigest("$2a$10$test").
		Build()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// 大文字で同じメールアドレスを試行
	_, err := validator.Validate(ctx, SignUpCreateValidatorInput{
		Email: "EXISTING@EXAMPLE.COM",
	})

	// メールアドレスの重複は大文字小文字を区別しないので、エラーになるべき
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Error("大文字小文字を区別しない重複チェックが必要です")
	}
}
