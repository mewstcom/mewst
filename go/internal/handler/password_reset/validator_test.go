package password_reset

import (
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates"
)

func TestCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         CreateValidatorInput
		wantErrors    bool
		expectedField string
	}{
		{
			name: "正常系: 有効なメールアドレス",
			input: CreateValidatorInput{
				Email: "user@example.com",
			},
			wantErrors: false,
		},
		{
			name: "正常系: サブドメインを含むメールアドレス",
			input: CreateValidatorInput{
				Email: "user@mail.example.com",
			},
			wantErrors: false,
		},
		{
			name: "正常系: 日本語ドメインのメールアドレス",
			input: CreateValidatorInput{
				Email: "user@example.co.jp",
			},
			wantErrors: false,
		},
		{
			name: "正常系: プラス記号を含むメールアドレス",
			input: CreateValidatorInput{
				Email: "user+tag@example.com",
			},
			wantErrors: false,
		},
		{
			name: "正常系: ドットを含むローカルパート",
			input: CreateValidatorInput{
				Email: "user.name@example.com",
			},
			wantErrors: false,
		},
		{
			name: "異常系: メールアドレスが空",
			input: CreateValidatorInput{
				Email: "",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: @マークがない",
			input: CreateValidatorInput{
				Email: "userexample.com",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: ドメイン部分がない",
			input: CreateValidatorInput{
				Email: "user@",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: ローカルパート部分がない",
			input: CreateValidatorInput{
				Email: "@example.com",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: 無効な文字を含む",
			input: CreateValidatorInput{
				Email: "user name@example.com",
			},
			wantErrors:    true,
			expectedField: "email",
		},
	}

	validator := NewCreateValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = templates.WithLocale(ctx, "ja")

			result := validator.Validate(ctx, tt.input)

			if tt.wantErrors {
				if !result.FormErrors.HasErrors() {
					t.Error("エラーが期待されたが、エラーがありません")
				}
				if tt.expectedField != "" && !result.FormErrors.HasFieldError(tt.expectedField) {
					t.Errorf("フィールド %q のエラーが期待されましたが、ありません", tt.expectedField)
				}
			} else {
				if result.FormErrors.HasErrors() {
					t.Errorf("エラーが期待されなかったが、エラーがあります: %v", result.FormErrors)
				}
			}
		})
	}
}

func TestCreateValidator_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	validator := NewCreateValidator()

	t.Run("メールアドレス必須エラーメッセージ", func(t *testing.T) {
		t.Parallel()

		input := CreateValidatorInput{Email: ""}
		result := validator.Validate(ctx, input)

		if !result.FormErrors.HasFieldError("email") {
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

		input := CreateValidatorInput{Email: "invalid-email"}
		result := validator.Validate(ctx, input)

		if !result.FormErrors.HasFieldError("email") {
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
