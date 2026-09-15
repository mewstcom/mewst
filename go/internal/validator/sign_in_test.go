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
			user, err := v.Validate(ctx, SignInCreateValidatorInput{
				Email:    tt.email,
				Password: tt.password,
			})

			if user != nil {
				t.Error("バリデーションエラー時のユーザー = 非nil、nilを期待")
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("エラー = nil、ValidationErrorを期待")
			}
			if tt.wantFieldError != "" && !ve.HasFieldError(tt.wantFieldError) {
				t.Errorf("%sのフィールドエラーを期待したが、無かった", tt.wantFieldError)
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

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	passwordDigest, _ := auth.HashPassword("password123")
	testutil.NewUserBuilder(t, tx).
		WithEmail("validator-signin-success@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	v := NewSignInCreateValidator(userRepo)

	user, err := v.Validate(ctx, SignInCreateValidatorInput{
		Email:    "validator-signin-success@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("Validate()のエラー = %v", err)
	}
	if user == nil {
		t.Fatal("Validate()のuser = nil、非nilを期待")
	}
	if user.Email != "validator-signin-success@example.com" {
		t.Errorf("Validate()のuser.Email = %v、期待値 = %v", user.Email, "validator-signin-success@example.com")
	}
}

func TestSignInCreateValidator_Validate_UserNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	v := NewSignInCreateValidator(userRepo)

	user, err := v.Validate(ctx, SignInCreateValidatorInput{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})

	if user != nil {
		t.Error("ユーザー = 非nil、nilを期待")
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("エラー = nil、ValidationErrorを期待")
	}
	if len(ve.Global) == 0 {
		t.Error("グローバルエラーを期待したが、無かった")
	}
}

func TestSignInCreateValidator_Validate_InvalidPassword(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	passwordDigest, _ := auth.HashPassword("correctpassword")
	testutil.NewUserBuilder(t, tx).
		WithEmail("validator-signin-invalid-password@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	v := NewSignInCreateValidator(userRepo)

	user, err := v.Validate(ctx, SignInCreateValidatorInput{
		Email:    "validator-signin-invalid-password@example.com",
		Password: "wrongpassword",
	})

	if user != nil {
		t.Error("ユーザー = 非nil、nilを期待")
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("エラー = nil、ValidationErrorを期待")
	}
	if len(ve.Global) == 0 {
		t.Error("グローバルエラーを期待したが、無かった")
	}
}

func TestSignInCreateValidator_Validate_ErrorMessageIsGeneric(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	passwordDigest, _ := auth.HashPassword("correctpassword")
	testutil.NewUserBuilder(t, tx).
		WithEmail("validator-signin-generic-message@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	v := NewSignInCreateValidator(userRepo)

	t.Run("ユーザーが存在しない場合も同じエラーメッセージ", func(t *testing.T) {
		t.Parallel()

		_, err1 := v.Validate(ctx, SignInCreateValidatorInput{
			Email:    "nonexistent@example.com",
			Password: "anypassword",
		})
		ve1 := model.AsValidationError(err1)
		if ve1 == nil || len(ve1.Global) == 0 {
			t.Fatal("グローバルエラーのメッセージを期待したが、無かった")
		}
		notFoundMsg := ve1.Global[0]

		_, err2 := v.Validate(ctx, SignInCreateValidatorInput{
			Email:    "validator-signin-generic-message@example.com",
			Password: "wrongpassword",
		})
		ve2 := model.AsValidationError(err2)
		if ve2 == nil || len(ve2.Global) == 0 {
			t.Fatal("グローバルエラーのメッセージを期待したが、無かった")
		}
		wrongPasswordMsg := ve2.Global[0]

		// セキュリティ上、両方のエラーメッセージが同じであることを確認
		if notFoundMsg != wrongPasswordMsg {
			t.Errorf("エラーメッセージが異なります: ユーザーが存在しない = %q、パスワードの誤り = %q", notFoundMsg, wrongPasswordMsg)
		}
	})
}

func TestSignInCreateValidator_Validate_GlobalError(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	v := NewSignInCreateValidator(userRepo)

	_, err := v.Validate(ctx, SignInCreateValidatorInput{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("ValidationErrorを期待したが、得られなかった")
	}

	// グローバルエラーとして返されることを確認 (フィールドエラーではない)
	if len(ve.Global) == 0 {
		t.Error("フィールドエラーではなくグローバルエラーを期待")
	}
	if len(ve.Fields) > 0 {
		t.Error("資格情報の検証ではフィールドエラーが無いことを期待したが、あった")
	}
}

func TestSignInCreateValidator_Validate_ValidEmailFormats(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
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
			// 親テストのtxを共有しているため、サブテスト間で並行にDBクエリを発行すると
			// lib/pqの接続状態が壊れてRollbackが "driver: bad connection" で失敗する。
			// サブテストは順次実行する。
			_, err := v.Validate(ctx, SignInCreateValidatorInput{
				Email:    tt.email,
				Password: "password123",
			})

			// 形式バリデーションでエラーにならないことを確認
			// (ユーザーが存在しないためグローバルエラーは発生するが、フィールドエラーは発生しない)
			ve := model.AsValidationError(err)
			if ve != nil && ve.HasFieldError("email") {
				t.Errorf("email形式バリデーションでエラーが発生: %v", ve.GetFieldErrors("email"))
			}
		})
	}
}
