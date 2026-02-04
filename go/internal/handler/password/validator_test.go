package password

import (
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates"
)

func TestUpdateValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         UpdateValidatorInput
		wantErrors    bool
		expectedField string
	}{
		{
			name: "正常系: 有効なパスワード（8文字）",
			input: UpdateValidatorInput{
				Password: "password",
			},
			wantErrors: false,
		},
		{
			name: "正常系: 有効なパスワード（長いパスワード）",
			input: UpdateValidatorInput{
				Password: "thisisaverylongpassword123!",
			},
			wantErrors: false,
		},
		{
			name: "正常系: 72バイトちょうどのパスワード",
			input: UpdateValidatorInput{
				Password: strings.Repeat("a", 72),
			},
			wantErrors: false,
		},
		{
			name: "正常系: 記号を含むパスワード",
			input: UpdateValidatorInput{
				Password: "P@ssw0rd!",
			},
			wantErrors: false,
		},
		{
			name: "異常系: パスワードが空",
			input: UpdateValidatorInput{
				Password: "",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: パスワードが短すぎる（7文字）",
			input: UpdateValidatorInput{
				Password: "passwor",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: パスワードが短すぎる（1文字）",
			input: UpdateValidatorInput{
				Password: "a",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: パスワードが長すぎる（73バイト）",
			input: UpdateValidatorInput{
				Password: strings.Repeat("a", 73),
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "正常系: 日本語を含むパスワード（8文字以上）",
			input: UpdateValidatorInput{
				Password: "パスワード123",
			},
			wantErrors: false,
		},
		{
			name: "異常系: 日本語のみで8文字未満（文字数カウント）",
			input: UpdateValidatorInput{
				Password: "パスワード12",
			},
			wantErrors:    true,
			expectedField: "password",
		},
	}

	validator := NewUpdateValidator()

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

func TestUpdateValidator_Validate_BoundaryValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	validator := NewUpdateValidator()

	t.Run("境界値: 7文字（NG）", func(t *testing.T) {
		t.Parallel()

		input := UpdateValidatorInput{Password: "1234567"}
		result := validator.Validate(ctx, input)

		if !result.FormErrors.HasFieldError("password") {
			t.Error("7文字のパスワードはエラーになるべき")
		}
	})

	t.Run("境界値: 8文字（OK）", func(t *testing.T) {
		t.Parallel()

		input := UpdateValidatorInput{Password: "12345678"}
		result := validator.Validate(ctx, input)

		if result.FormErrors.HasErrors() {
			t.Error("8文字のパスワードはエラーにならないべき")
		}
	})

	t.Run("境界値: 72バイト（OK）", func(t *testing.T) {
		t.Parallel()

		input := UpdateValidatorInput{Password: strings.Repeat("a", 72)}
		result := validator.Validate(ctx, input)

		if result.FormErrors.HasErrors() {
			t.Error("72バイトのパスワードはエラーにならないべき")
		}
	})

	t.Run("境界値: 73バイト（NG）", func(t *testing.T) {
		t.Parallel()

		input := UpdateValidatorInput{Password: strings.Repeat("a", 73)}
		result := validator.Validate(ctx, input)

		if !result.FormErrors.HasFieldError("password") {
			t.Error("73バイトのパスワードはエラーになるべき")
		}
	})
}
