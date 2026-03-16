package sign_up

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestCreateValidator_EmptyEmail(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, CreateValidatorInput{
		Email: "",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if !result.FormErrors.HasErrors() {
		t.Error("バリデーションエラーが発生するべきです")
	}

	emailErrors := result.FormErrors.Fields["email"]
	if len(emailErrors) == 0 {
		t.Error("emailフィールドのエラーが発生するべきです")
	}
}

func TestCreateValidator_InvalidEmailFormat(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	invalidEmails := []string{
		"invalid",
		"invalid@",
		"@example.com",
		"invalid@.com",
	}

	for _, email := range invalidEmails {
		t.Run(email, func(t *testing.T) {
			result := validator.Validate(ctx, CreateValidatorInput{
				Email: email,
			})

			if result.Err != nil {
				t.Fatalf("予期しないエラー: %v", result.Err)
			}

			if !result.FormErrors.HasErrors() {
				t.Error("バリデーションエラーが発生するべきです")
			}

			emailErrors := result.FormErrors.Fields["email"]
			if len(emailErrors) == 0 {
				t.Error("emailフィールドのエラーが発生するべきです")
			}
		})
	}
}

func TestCreateValidator_ValidEmail(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	validEmails := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user@subdomain.example.com",
	}

	for _, email := range validEmails {
		t.Run(email, func(t *testing.T) {
			result := validator.Validate(ctx, CreateValidatorInput{
				Email: email,
			})

			if result.Err != nil {
				t.Fatalf("予期しないエラー: %v", result.Err)
			}

			if result.FormErrors.HasErrors() {
				t.Errorf("バリデーションエラーが発生すべきではありません: %v", result.FormErrors.Fields)
			}
		})
	}
}

func TestCreateValidator_EmailAlreadyTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	// 既存のユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("existing@example.com").
		WithPasswordDigest("$2a$10$test").
		Build()

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, CreateValidatorInput{
		Email: "existing@example.com",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if !result.FormErrors.HasErrors() {
		t.Error("バリデーションエラーが発生するべきです")
	}

	emailErrors := result.FormErrors.Fields["email"]
	if len(emailErrors) == 0 {
		t.Error("emailフィールドのエラーが発生するべきです")
	}
}

func TestCreateValidator_EmailNotTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, CreateValidatorInput{
		Email: "newuser@example.com",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if result.FormErrors.HasErrors() {
		t.Errorf("バリデーションエラーが発生すべきではありません: %v", result.FormErrors.Fields)
	}
}

func TestCreateValidator_CaseInsensitiveEmail(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	// 小文字のメールアドレスで既存ユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("existing@example.com").
		WithPasswordDigest("$2a$10$test").
		Build()

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	// 大文字で同じメールアドレスを試行
	result := validator.Validate(ctx, CreateValidatorInput{
		Email: "EXISTING@EXAMPLE.COM",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	// メールアドレスの重複は大文字小文字を区別しないので、エラーになるべき
	if !result.FormErrors.HasErrors() {
		t.Error("大文字小文字を区別しない重複チェックが必要です")
	}
}
