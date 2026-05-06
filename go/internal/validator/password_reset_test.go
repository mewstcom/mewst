package validator

import (
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
)

func TestPasswordResetCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         PasswordResetCreateValidatorInput
		wantErrors    bool
		expectedField string
	}{
		{
			name: "正常系: 有効なメールアドレス",
			input: PasswordResetCreateValidatorInput{
				Email: "user@example.com",
			},
			wantErrors: false,
		},
		{
			name: "正常系: サブドメインを含むメールアドレス",
			input: PasswordResetCreateValidatorInput{
				Email: "user@mail.example.com",
			},
			wantErrors: false,
		},
		{
			name: "正常系: 日本語ドメインのメールアドレス",
			input: PasswordResetCreateValidatorInput{
				Email: "user@example.co.jp",
			},
			wantErrors: false,
		},
		{
			name: "正常系: プラス記号を含むメールアドレス",
			input: PasswordResetCreateValidatorInput{
				Email: "user+tag@example.com",
			},
			wantErrors: false,
		},
		{
			name: "正常系: ドットを含むローカルパート",
			input: PasswordResetCreateValidatorInput{
				Email: "user.name@example.com",
			},
			wantErrors: false,
		},
		{
			name: "異常系: メールアドレスが空",
			input: PasswordResetCreateValidatorInput{
				Email: "",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: @マークがない",
			input: PasswordResetCreateValidatorInput{
				Email: "userexample.com",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: ドメイン部分がない",
			input: PasswordResetCreateValidatorInput{
				Email: "user@",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: ローカルパート部分がない",
			input: PasswordResetCreateValidatorInput{
				Email: "@example.com",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: 無効な文字を含む",
			input: PasswordResetCreateValidatorInput{
				Email: "user name@example.com",
			},
			wantErrors:    true,
			expectedField: "email",
		},
	}

	v := NewPasswordResetCreateValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			err := v.Validate(ctx, tt.input)

			if tt.wantErrors {
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Error("エラーが期待されたが、エラーがありません")
				}
				if tt.expectedField != "" && ve != nil && !ve.HasFieldError(tt.expectedField) {
					t.Errorf("フィールド %q のエラーが期待されましたが、ありません", tt.expectedField)
				}
			} else {
				if err != nil {
					t.Errorf("エラーが期待されなかったが、エラーがあります: %v", err)
				}
			}
		})
	}
}

func TestPasswordResetCreateValidator_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	v := NewPasswordResetCreateValidator()

	t.Run("メールアドレス必須エラーメッセージ", func(t *testing.T) {
		t.Parallel()

		input := PasswordResetCreateValidatorInput{Email: ""}
		err := v.Validate(ctx, input)

		ve := model.AsValidationError(err)
		if ve == nil || !ve.HasFieldError("email") {
			t.Fatal("emailフィールドにエラーがありません")
		}

		emailErrors := ve.GetFieldErrors("email")
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

		input := PasswordResetCreateValidatorInput{Email: "invalid-email"}
		err := v.Validate(ctx, input)

		ve := model.AsValidationError(err)
		if ve == nil || !ve.HasFieldError("email") {
			t.Fatal("emailフィールドにエラーがありません")
		}

		emailErrors := ve.GetFieldErrors("email")
		if len(emailErrors) == 0 {
			t.Fatal("emailエラーが空です")
		}

		// メール形式エラーであることを確認
		if !strings.Contains(emailErrors[0], "メール") && !strings.Contains(emailErrors[0], "形式") {
			t.Errorf("メール形式エラーメッセージが期待されましたが、取得: %s", emailErrors[0])
		}
	})
}
