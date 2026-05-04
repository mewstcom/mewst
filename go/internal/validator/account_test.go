package validator

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestAccountCreateValidator_EmptyAtname(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	err := validator.Validate(ctx, AccountCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "",
		Password: "password123",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("バリデーションエラーが発生するべきです")
	}

	atnameErrors := ve.GetFieldErrors("atname")
	if len(atnameErrors) == 0 {
		t.Error("atnameフィールドのエラーが発生するべきです")
	}
}

func TestAccountCreateValidator_InvalidAtnameFormat(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	invalidAtnames := []string{
		"user name",
		"user@name",
		"user.name",
		"ユーザー",
		"user-name",
	}

	for _, atname := range invalidAtnames {
		err := validator.Validate(ctx, AccountCreateValidatorInput{
			Email:    "test@example.com",
			Atname:   atname,
			Password: "password123",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Errorf("バリデーションエラーが発生するべきです (atname=%s)", atname)
			continue
		}

		atnameErrors := ve.GetFieldErrors("atname")
		if len(atnameErrors) == 0 {
			t.Errorf("atnameフィールドのエラーが発生するべきです (atname=%s)", atname)
		}
	}
}

func TestAccountCreateValidator_AtnameTooLong(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	err := validator.Validate(ctx, AccountCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "a23456789012345678901", // 21文字
		Password: "password123",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("バリデーションエラーが発生するべきです")
	}

	atnameErrors := ve.GetFieldErrors("atname")
	if len(atnameErrors) == 0 {
		t.Error("atnameフィールドのエラーが発生するべきです")
	}
}

func TestAccountCreateValidator_ReservedAtname(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	reservedNames := []string{
		"admin",
		"Admin",
		"ADMIN",
		"support",
		"mewst",
	}

	for _, atname := range reservedNames {
		err := validator.Validate(ctx, AccountCreateValidatorInput{
			Email:    "test@example.com",
			Atname:   atname,
			Password: "password123",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Errorf("バリデーションエラーが発生するべきです (atname=%s)", atname)
			continue
		}

		atnameErrors := ve.GetFieldErrors("atname")
		if len(atnameErrors) == 0 {
			t.Errorf("atnameフィールドのエラーが発生するべきです (atname=%s)", atname)
		}
	}
}

func TestAccountCreateValidator_AtnameAlreadyTaken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	// 既存のプロフィールを作成
	testutil.NewProfileBuilder(t, tx).
		WithAtname("existinguser").
		Build()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	err := validator.Validate(ctx, AccountCreateValidatorInput{
		Email:    "accounts-atname-taken@example.com",
		Atname:   "existinguser",
		Password: "password123",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("バリデーションエラーが発生するべきです")
	}

	atnameErrors := ve.GetFieldErrors("atname")
	if len(atnameErrors) == 0 {
		t.Error("atnameフィールドのエラーが発生するべきです")
	}
}

func TestAccountCreateValidator_EmailAlreadyTaken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	// 既存のユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("existing@example.com").
		WithPasswordDigest("$2a$10$test").
		Build()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	err := validator.Validate(ctx, AccountCreateValidatorInput{
		Email:    "existing@example.com",
		Atname:   "newuser",
		Password: "password123",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("バリデーションエラーが発生するべきです")
	}

	// メールアドレスは編集不可のため、グローバルエラーとして表示される
	if len(ve.Global) == 0 {
		t.Error("グローバルエラーが発生するべきです")
	}
}

func TestAccountCreateValidator_EmptyPassword(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	err := validator.Validate(ctx, AccountCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "testuser",
		Password: "",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("バリデーションエラーが発生するべきです")
	}

	passwordErrors := ve.GetFieldErrors("password")
	if len(passwordErrors) == 0 {
		t.Error("passwordフィールドのエラーが発生するべきです")
	}
}

func TestAccountCreateValidator_PasswordTooShort(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	err := validator.Validate(ctx, AccountCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "testuser",
		Password: "short",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("バリデーションエラーが発生するべきです")
	}

	passwordErrors := ve.GetFieldErrors("password")
	if len(passwordErrors) == 0 {
		t.Error("passwordフィールドのエラーが発生するべきです")
	}
}

func TestAccountCreateValidator_PasswordTooLong(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// 73文字のパスワード（bcryptの72文字制限を超える）
	longPassword := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"

	err := validator.Validate(ctx, AccountCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "testuser",
		Password: longPassword,
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("バリデーションエラーが発生するべきです")
	}

	passwordErrors := ve.GetFieldErrors("password")
	if len(passwordErrors) == 0 {
		t.Error("passwordフィールドのエラーが発生するべきです")
	}
}

func TestAccountCreateValidator_ValidInput(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	err := validator.Validate(ctx, AccountCreateValidatorInput{
		Email:    "accounts-valid-input@example.com",
		Atname:   "accountsvalid",
		Password: "password123",
	})

	if err != nil {
		t.Errorf("バリデーションエラーが発生すべきではありません: %v", err)
	}
}

func TestAccountCreateValidator_ValidAtnameFormats(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))
	validator := NewAccountCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	validAtnames := []string{
		"user123",
		"User_Name",
		"test",
		"a",
		"12345678901234567890", // ちょうど20文字
	}

	for _, atname := range validAtnames {
		err := validator.Validate(ctx, AccountCreateValidatorInput{
			Email:    "test@example.com",
			Atname:   atname,
			Password: "password123",
		})

		if err != nil {
			t.Errorf("バリデーションエラーが発生すべきではありません (atname=%s): %v", atname, err)
		}
	}
}
