package validator

import (
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestSignInCreateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		email          string
		password       string
		wantFieldError string
	}{
		{
			name:           "異常系: メールアドレスが空",
			email:          "",
			password:       "password123",
			wantFieldError: "email",
		},
		{
			name:           "異常系: パスワードが空",
			email:          "user@example.com",
			password:       "",
			wantFieldError: "password",
		},
		{
			name:           "異常系: 両方が空",
			email:          "",
			password:       "",
			wantFieldError: "email",
		},
		{
			name:           "異常系: 無効なメールアドレス形式",
			email:          "invalid-email",
			password:       "password123",
			wantFieldError: "email",
		},
		{
			name:           "異常系: @マークがないメールアドレス",
			email:          "userexample.com",
			password:       "password123",
			wantFieldError: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			v := NewSignInCreateValidator(nil)
			output, err := v.Validate(ctx, SignInCreateValidatorInput{
				Email:    tt.email,
				Password: tt.password,
			})

			if output != nil {
				t.Error("expected nil output for validation error")
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("expected ValidationError, got nil")
			}
			if tt.wantFieldError != "" && !ve.HasFieldError(tt.wantFieldError) {
				t.Errorf("expected field error for %s, but not found", tt.wantFieldError)
			}
		})
	}
}

func TestSignInCreateValidator_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	t.Run("メールアドレス必須エラーメッセージ", func(t *testing.T) {
		t.Parallel()

		v := NewSignInCreateValidator(nil)
		_, err := v.Validate(ctx, SignInCreateValidatorInput{
			Email:    "",
			Password: "password123",
		})

		ve := model.AsValidationError(err)
		if ve == nil || !ve.HasFieldError("email") {
			t.Fatal("emailフィールドにエラーがありません")
		}

		emailErrors := ve.GetFieldErrors("email")
		if len(emailErrors) == 0 {
			t.Fatal("emailエラーが空です")
		}

		if emailErrors[0] == "" {
			t.Error("エラーメッセージが空です")
		}
	})

	t.Run("メールアドレス形式エラーメッセージ", func(t *testing.T) {
		t.Parallel()

		v := NewSignInCreateValidator(nil)
		_, err := v.Validate(ctx, SignInCreateValidatorInput{
			Email:    "invalid-email",
			Password: "password123",
		})

		ve := model.AsValidationError(err)
		if ve == nil || !ve.HasFieldError("email") {
			t.Fatal("emailフィールドにエラーがありません")
		}

		emailErrors := ve.GetFieldErrors("email")
		if len(emailErrors) == 0 {
			t.Fatal("emailエラーが空です")
		}

		if !strings.Contains(emailErrors[0], "メール") && !strings.Contains(emailErrors[0], "形式") {
			t.Errorf("メール形式エラーメッセージが期待されましたが、取得: %s", emailErrors[0])
		}
	})
}

func TestSignInCreateValidator_Validate_Success(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	passwordDigest, _ := auth.HashPassword("password123")
	testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	v := NewSignInCreateValidator(userRepo)

	output, err := v.Validate(ctx, SignInCreateValidatorInput{
		Email:    "test@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if output == nil {
		t.Fatal("Validate() output = nil, want non-nil")
	}
	if output.User == nil {
		t.Error("Validate() output.User = nil, want non-nil")
	}
	if output.User != nil && output.User.Email != "test@example.com" {
		t.Errorf("Validate() output.User.Email = %v, want %v", output.User.Email, "test@example.com")
	}
}

func TestSignInCreateValidator_Validate_UserNotFound(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	v := NewSignInCreateValidator(userRepo)

	output, err := v.Validate(ctx, SignInCreateValidatorInput{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})

	if output != nil {
		t.Error("expected nil output")
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if len(ve.Global) == 0 {
		t.Error("expected global error")
	}
}

func TestSignInCreateValidator_Validate_InvalidPassword(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	passwordDigest, _ := auth.HashPassword("correctpassword")
	testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	v := NewSignInCreateValidator(userRepo)

	output, err := v.Validate(ctx, SignInCreateValidatorInput{
		Email:    "test@example.com",
		Password: "wrongpassword",
	})

	if output != nil {
		t.Error("expected nil output")
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if len(ve.Global) == 0 {
		t.Error("expected global error")
	}
}

func TestSignInCreateValidator_Validate_ErrorMessageIsGeneric(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	passwordDigest, _ := auth.HashPassword("correctpassword")
	testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	v := NewSignInCreateValidator(userRepo)

	t.Run("ユーザーが存在しない場合も同じエラーメッセージ", func(t *testing.T) {
		t.Parallel()

		_, err1 := v.Validate(ctx, SignInCreateValidatorInput{
			Email:    "nonexistent@example.com",
			Password: "anypassword",
		})
		ve1 := model.AsValidationError(err1)
		if ve1 == nil || len(ve1.Global) == 0 {
			t.Fatal("expected global error message")
		}
		notFoundMsg := ve1.Global[0]

		_, err2 := v.Validate(ctx, SignInCreateValidatorInput{
			Email:    "test@example.com",
			Password: "wrongpassword",
		})
		ve2 := model.AsValidationError(err2)
		if ve2 == nil || len(ve2.Global) == 0 {
			t.Fatal("expected global error message")
		}
		wrongPasswordMsg := ve2.Global[0]

		// セキュリティ上、両方のエラーメッセージが同じであることを確認
		if notFoundMsg != wrongPasswordMsg {
			t.Errorf("エラーメッセージが異なります: user not found = %q, wrong password = %q", notFoundMsg, wrongPasswordMsg)
		}
	})
}

func TestSignInCreateValidator_Validate_GlobalError(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	v := NewSignInCreateValidator(userRepo)

	_, err := v.Validate(ctx, SignInCreateValidatorInput{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError")
	}

	// グローバルエラーとして返されることを確認（フィールドエラーではない）
	if len(ve.Global) == 0 {
		t.Error("expected global error, not field error")
	}
	if len(ve.Fields) > 0 {
		t.Error("should not have field errors for credential validation")
	}
}

func TestSignInCreateValidator_Validate_ValidEmailFormats(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(db).WithTx(tx)
	v := NewSignInCreateValidator(userRepo)

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

			_, err := v.Validate(ctx, SignInCreateValidatorInput{
				Email:    tt.email,
				Password: "password123",
			})

			// 形式バリデーションでエラーにならないことを確認
			// （ユーザーが存在しないためグローバルエラーは発生するが、フィールドエラーは発生しない）
			ve := model.AsValidationError(err)
			if ve != nil && ve.HasFieldError("email") {
				t.Errorf("email形式バリデーションでエラーが発生: %v", ve.GetFieldErrors("email"))
			}
		})
	}
}
