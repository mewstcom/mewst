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
		validator     CreateValidator
		wantErrors    bool
		expectedField string
	}{
		{
			name: "正常系: 有効なメールアドレス",
			validator: CreateValidator{
				Email: "user@example.com",
			},
			wantErrors: false,
		},
		{
			name: "正常系: サブドメインを含むメールアドレス",
			validator: CreateValidator{
				Email: "user@mail.example.com",
			},
			wantErrors: false,
		},
		{
			name: "正常系: 日本語ドメインのメールアドレス",
			validator: CreateValidator{
				Email: "user@example.co.jp",
			},
			wantErrors: false,
		},
		{
			name: "正常系: プラス記号を含むメールアドレス",
			validator: CreateValidator{
				Email: "user+tag@example.com",
			},
			wantErrors: false,
		},
		{
			name: "正常系: ドットを含むローカルパート",
			validator: CreateValidator{
				Email: "user.name@example.com",
			},
			wantErrors: false,
		},
		{
			name: "異常系: メールアドレスが空",
			validator: CreateValidator{
				Email: "",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: @マークがない",
			validator: CreateValidator{
				Email: "userexample.com",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: ドメイン部分がない",
			validator: CreateValidator{
				Email: "user@",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: ローカルパート部分がない",
			validator: CreateValidator{
				Email: "@example.com",
			},
			wantErrors:    true,
			expectedField: "email",
		},
		{
			name: "異常系: 無効な文字を含む",
			validator: CreateValidator{
				Email: "user name@example.com",
			},
			wantErrors:    true,
			expectedField: "email",
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

func TestCreateValidator_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	t.Run("メールアドレス必須エラーメッセージ", func(t *testing.T) {
		t.Parallel()

		validator := CreateValidator{Email: ""}
		formErrors := validator.Validate(ctx)

		if !formErrors.HasFieldError("email") {
			t.Fatal("emailフィールドにエラーがありません")
		}

		emailErrors := formErrors.GetFieldErrors("email")
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

		validator := CreateValidator{Email: "invalid-email"}
		formErrors := validator.Validate(ctx)

		if !formErrors.HasFieldError("email") {
			t.Fatal("emailフィールドにエラーがありません")
		}

		emailErrors := formErrors.GetFieldErrors("email")
		if len(emailErrors) == 0 {
			t.Fatal("emailエラーが空です")
		}

		// メール形式エラーであることを確認
		if !strings.Contains(emailErrors[0], "メール") && !strings.Contains(emailErrors[0], "形式") {
			t.Errorf("メール形式エラーメッセージが期待されましたが、取得: %s", emailErrors[0])
		}
	})
}
