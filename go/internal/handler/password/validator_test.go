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
		validator     UpdateValidator
		wantErrors    bool
		expectedField string
	}{
		{
			name: "正常系: 有効なパスワード（8文字）",
			validator: UpdateValidator{
				Password: "password",
			},
			wantErrors: false,
		},
		{
			name: "正常系: 有効なパスワード（長いパスワード）",
			validator: UpdateValidator{
				Password: "thisisaverylongpassword123!",
			},
			wantErrors: false,
		},
		{
			name: "正常系: 72バイトちょうどのパスワード",
			validator: UpdateValidator{
				Password: strings.Repeat("a", 72),
			},
			wantErrors: false,
		},
		{
			name: "正常系: 記号を含むパスワード",
			validator: UpdateValidator{
				Password: "P@ssw0rd!",
			},
			wantErrors: false,
		},
		{
			name: "異常系: パスワードが空",
			validator: UpdateValidator{
				Password: "",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: パスワードが短すぎる（7文字）",
			validator: UpdateValidator{
				Password: "passwor",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: パスワードが短すぎる（1文字）",
			validator: UpdateValidator{
				Password: "a",
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "異常系: パスワードが長すぎる（73バイト）",
			validator: UpdateValidator{
				Password: strings.Repeat("a", 73),
			},
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name: "正常系: 日本語を含むパスワード（8文字以上）",
			validator: UpdateValidator{
				Password: "パスワード123",
			},
			wantErrors: false,
		},
		{
			name: "異常系: 日本語のみで8文字未満（文字数カウント）",
			validator: UpdateValidator{
				Password: "パスワード12",
			},
			wantErrors:    true,
			expectedField: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = templates.WithLocale(ctx, "ja")

			formErrors := tt.validator.Validate(ctx)

			if tt.wantErrors {
				if !formErrors.HasErrors() {
					t.Error("エラーが期待されたが、エラーがありません")
				}
				if tt.expectedField != "" && !formErrors.HasFieldError(tt.expectedField) {
					t.Errorf("フィールド %q のエラーが期待されましたが、ありません", tt.expectedField)
				}
			} else {
				if formErrors.HasErrors() {
					t.Errorf("エラーが期待されなかったが、エラーがあります: %v", formErrors)
				}
			}
		})
	}
}

func TestUpdateValidator_Validate_BoundaryValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	t.Run("境界値: 7文字（NG）", func(t *testing.T) {
		t.Parallel()

		validator := UpdateValidator{Password: "1234567"}
		formErrors := validator.Validate(ctx)

		if !formErrors.HasFieldError("password") {
			t.Error("7文字のパスワードはエラーになるべき")
		}
	})

	t.Run("境界値: 8文字（OK）", func(t *testing.T) {
		t.Parallel()

		validator := UpdateValidator{Password: "12345678"}
		formErrors := validator.Validate(ctx)

		if formErrors.HasErrors() {
			t.Error("8文字のパスワードはエラーにならないべき")
		}
	})

	t.Run("境界値: 72バイト（OK）", func(t *testing.T) {
		t.Parallel()

		validator := UpdateValidator{Password: strings.Repeat("a", 72)}
		formErrors := validator.Validate(ctx)

		if formErrors.HasErrors() {
			t.Error("72バイトのパスワードはエラーにならないべき")
		}
	})

	t.Run("境界値: 73バイト（NG）", func(t *testing.T) {
		t.Parallel()

		validator := UpdateValidator{Password: strings.Repeat("a", 73)}
		formErrors := validator.Validate(ctx)

		if !formErrors.HasFieldError("password") {
			t.Error("73バイトのパスワードはエラーになるべき")
		}
	})
}
