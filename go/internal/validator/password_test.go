package validator

import (
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates"
)

func TestPasswordUpdateValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         PasswordUpdateValidatorInput
		wantErrors    bool
		expectedField string
	}{
		{
			name: "正常系: 有効なパスワード（8文字）",
			input: PasswordUpdateValidatorInput{
				Password: "password",
			},
			wantErrors: false,
		},
		{
			name: "正常系: 有効なパスワード（長いパスワード）",
			input: PasswordUpdateValidatorInput{
				Password: "thisisaverylongpassword123!",
			},
			wantErrors: false,
		},
		{
			name: "正常系: 72バイトちょうどのパスワード",
			input: PasswordUpdateValidatorInput{
				Password: strings.Repeat("a", 72),
			},
			wantErrors: false,
		},
		{
			name: "正常系: 記号を含むパスワード",
			input: PasswordUpdateValidatorInput{
				Password: "P@ssw0rd!",
			},
			wantErrors: false,
		},
		{
			name: "異常系: パスワードが空",
			input: PasswordUpdateValidatorInput{
				Password: "",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: パスワードが短すぎる（7文字）",
			input: PasswordUpdateValidatorInput{
				Password: "passwor",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: パスワードが短すぎる（1文字）",
			input: PasswordUpdateValidatorInput{
				Password: "a",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: パスワードが長すぎる（73バイト）",
			input: PasswordUpdateValidatorInput{
				Password: strings.Repeat("a", 73),
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "正常系: 日本語を含むパスワード（8文字以上）",
			input: PasswordUpdateValidatorInput{
				Password: "パスワード123",
			},
			wantErrors: false,
		},
		{
			name: "異常系: 日本語のみで8文字未満（文字数カウント）",
			input: PasswordUpdateValidatorInput{
				Password: "パスワード12",
			},
			wantErrors:    true,
			expectedField: "password",
		},
	}

	v := NewPasswordUpdateValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = templates.WithLocale(ctx, "ja")

			result := v.Validate(ctx, tt.input)

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

func TestPasswordUpdateValidator_Validate_BoundaryValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	v := NewPasswordUpdateValidator()

	t.Run("境界値: 7文字（NG）", func(t *testing.T) {
		t.Parallel()

		input := PasswordUpdateValidatorInput{Password: "1234567"}
		result := v.Validate(ctx, input)

		if !result.FormErrors.HasFieldError("password") {
			t.Error("7文字のパスワードはエラーになるべき")
		}
	})

	t.Run("境界値: 8文字（OK）", func(t *testing.T) {
		t.Parallel()

		input := PasswordUpdateValidatorInput{Password: "12345678"}
		result := v.Validate(ctx, input)

		if result.FormErrors.HasErrors() {
			t.Error("8文字のパスワードはエラーにならないべき")
		}
	})

	t.Run("境界値: 72バイト（OK）", func(t *testing.T) {
		t.Parallel()

		input := PasswordUpdateValidatorInput{Password: strings.Repeat("a", 72)}
		result := v.Validate(ctx, input)

		if result.FormErrors.HasErrors() {
			t.Error("72バイトのパスワードはエラーにならないべき")
		}
	})

	t.Run("境界値: 73バイト（NG）", func(t *testing.T) {
		t.Parallel()

		input := PasswordUpdateValidatorInput{Password: strings.Repeat("a", 73)}
		result := v.Validate(ctx, input)

		if !result.FormErrors.HasFieldError("password") {
			t.Error("73バイトのパスワードはエラーになるべき")
		}
	})
}
