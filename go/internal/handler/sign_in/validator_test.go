package sign_in

import (
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestCreateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	tests := []struct {
		name          string
		input         CreateValidatorInput
		wantErrors    bool
		expectedField string
	}{
		{
			name: "異常系: メールアドレスが空",
			input: CreateValidatorInput{
				Email:    "",
				Password: "password123",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: パスワードが空",
			input: CreateValidatorInput{
				Email:    "user@example.com",
				Password: "",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: 両方が空",
			input: CreateValidatorInput{
				Email:    "",
				Password: "",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: 無効なメールアドレス形式",
			input: CreateValidatorInput{
				Email:    "invalid-email",
				Password: "password123",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: @マークがないメールアドレス",
			input: CreateValidatorInput{
				Email:    "userexample.com",
				Password: "password123",
			},
			wantErrors:    true,
			expectedField: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := validator.Validate(ctx, tt.input)

			if tt.wantErrors {
				if result.FormErrors == nil || !result.FormErrors.HasErrors() {
					t.Error("エラーが期待されたが、エラーがありません")
				}
				if tt.expectedField != "" && !result.FormErrors.HasFieldError(tt.expectedField) {
					t.Errorf("フィールド %q のエラーが期待されましたが、ありません", tt.expectedField)
				}
			}
		})
	}
}

func TestCreateValidator_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	t.Run("メールアドレス必須エラーメッセージ", func(t *testing.T) {
		t.Parallel()

		input := CreateValidatorInput{
			Email:    "",
			Password: "password123",
		}
		result := validator.Validate(ctx, input)

		if result.FormErrors == nil || !result.FormErrors.HasFieldError("email") {
			t.Fatal("emailフィールドにエラーがありません")
		}

		emailErrors := result.FormErrors.GetFieldErrors("email")
		if len(emailErrors) == 0 {
			t.Fatal("emailエラーが空です")
		}

		// エラーメッセージが空でないことを確認
		if emailErrors[0] == "" {
			t.Error("エラーメッセージが空です")
		}
	})

	t.Run("メールアドレス形式エラーメッセージ", func(t *testing.T) {
		t.Parallel()

		input := CreateValidatorInput{
			Email:    "invalid-email",
			Password: "password123",
		}
		result := validator.Validate(ctx, input)

		if result.FormErrors == nil || !result.FormErrors.HasFieldError("email") {
			t.Fatal("emailフィールドにエラーがありません")
		}

		emailErrors := result.FormErrors.GetFieldErrors("email")
		if len(emailErrors) == 0 {
			t.Fatal("emailエラーが空です")
		}

		// メール形式エラーであることを確認
		if !strings.Contains(emailErrors[0], "メール") && !strings.Contains(emailErrors[0], "形式") {
			t.Errorf("メール形式エラーメッセージが期待されましたが、取得: %s", emailErrors[0])
		}
	})
}

func TestCreateValidator_Validate_Success(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	// テストユーザーを作成
	passwordDigest, _ := auth.HashPassword("password123")
	testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	input := CreateValidatorInput{
		Email:    "test@example.com",
		Password: "password123",
	}

	result := validator.Validate(ctx, input)

	if result.Err != nil {
		t.Fatalf("Validate() error = %v", result.Err)
	}
	if result.FormErrors != nil && result.FormErrors.HasErrors() {
		t.Errorf("Validate() formErrors = %v, want nil", result.FormErrors)
	}
	if result.User == nil {
		t.Error("Validate() user = nil, want non-nil")
	}
	if result.User != nil && result.User.Email != "test@example.com" {
		t.Errorf("Validate() user.Email = %v, want %v", result.User.Email, "test@example.com")
	}
}

func TestCreateValidator_Validate_UserNotFound(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	// ユーザーを作成しない

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	input := CreateValidatorInput{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	result := validator.Validate(ctx, input)

	if result.Err != nil {
		t.Fatalf("Validate() error = %v, want nil", result.Err)
	}
	if result.User != nil {
		t.Error("Validate() user should be nil for non-existent user")
	}
	if result.FormErrors == nil {
		t.Fatal("Validate() formErrors = nil, want non-nil")
	}
	if !result.FormErrors.HasErrors() {
		t.Error("Validate() formErrors should have errors")
	}
}

func TestCreateValidator_Validate_InvalidPassword(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	// テストユーザーを作成
	passwordDigest, _ := auth.HashPassword("correctpassword")
	testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	input := CreateValidatorInput{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	result := validator.Validate(ctx, input)

	if result.Err != nil {
		t.Fatalf("Validate() error = %v, want nil", result.Err)
	}
	if result.User != nil {
		t.Error("Validate() user should be nil for invalid password")
	}
	if result.FormErrors == nil {
		t.Fatal("Validate() formErrors = nil, want non-nil")
	}
	if !result.FormErrors.HasErrors() {
		t.Error("Validate() formErrors should have errors")
	}
}

func TestCreateValidator_Validate_ErrorMessageIsGeneric(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	// テストユーザーを作成
	passwordDigest, _ := auth.HashPassword("correctpassword")
	testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	t.Run("ユーザーが存在しない場合も同じエラーメッセージ", func(t *testing.T) {
		t.Parallel()

		input := CreateValidatorInput{
			Email:    "nonexistent@example.com",
			Password: "anypassword",
		}

		result := validator.Validate(ctx, input)

		if result.FormErrors == nil || len(result.FormErrors.Global) == 0 {
			t.Fatal("expected global error message")
		}

		notFoundMsg := result.FormErrors.Global[0]

		// パスワードが間違っている場合
		input2 := CreateValidatorInput{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		result2 := validator.Validate(ctx, input2)

		if result2.FormErrors == nil || len(result2.FormErrors.Global) == 0 {
			t.Fatal("expected global error message")
		}

		wrongPasswordMsg := result2.FormErrors.Global[0]

		// セキュリティ上、両方のエラーメッセージが同じであることを確認
		if notFoundMsg != wrongPasswordMsg {
			t.Errorf("エラーメッセージが異なります: user not found = %q, wrong password = %q", notFoundMsg, wrongPasswordMsg)
		}
	})
}

func TestCreateValidator_Validate_GlobalError(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	input := CreateValidatorInput{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	result := validator.Validate(ctx, input)

	if result.FormErrors == nil {
		t.Fatal("formErrors should not be nil")
	}

	// グローバルエラーとして返されることを確認（フィールドエラーではない）
	if len(result.FormErrors.Global) == 0 {
		t.Error("expected global error, not field error")
	}
	if len(result.FormErrors.Fields) > 0 {
		t.Error("should not have field errors for credential validation")
	}
}

func TestCreateValidator_Validate_ValidEmailFormats(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	validator := NewCreateValidator(userRepo)

	tests := []struct {
		name  string
		email string
	}{
		{
			name:  "日本語ドメインのメールアドレス",
			email: "user@example.co.jp",
		},
		{
			name:  "プラス記号を含むメールアドレス",
			email: "user+tag@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := CreateValidatorInput{
				Email:    tt.email,
				Password: "password123",
			}

			result := validator.Validate(ctx, input)

			// 形式バリデーションでエラーにならないことを確認
			// （ユーザーが存在しないためグローバルエラーは発生するが、フィールドエラーは発生しない）
			if result.FormErrors != nil && result.FormErrors.HasFieldError("email") {
				t.Errorf("email形式バリデーションでエラーが発生: %v", result.FormErrors.GetFieldErrors("email"))
			}
		})
	}
}
