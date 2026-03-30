package validator

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestAccountsCreateValidator_EmptyAtname(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, AccountsCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "",
		Password: "password123",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if !result.FormErrors.HasErrors() {
		t.Error("バリデーションエラーが発生するべきです")
	}

	atnameErrors := result.FormErrors.Fields["atname"]
	if len(atnameErrors) == 0 {
		t.Error("atnameフィールドのエラーが発生するべきです")
	}
}

func TestAccountsCreateValidator_InvalidAtnameFormat(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	invalidAtnames := []string{
		"user name",
		"user@name",
		"user.name",
		"ユーザー",
		"user-name",
	}

	for _, atname := range invalidAtnames {
		result := validator.Validate(ctx, AccountsCreateValidatorInput{
			Email:    "test@example.com",
			Atname:   atname,
			Password: "password123",
		})

		if result.Err != nil {
			t.Fatalf("予期しないエラー (atname=%s): %v", atname, result.Err)
		}

		if !result.FormErrors.HasErrors() {
			t.Errorf("バリデーションエラーが発生するべきです (atname=%s)", atname)
		}

		atnameErrors := result.FormErrors.Fields["atname"]
		if len(atnameErrors) == 0 {
			t.Errorf("atnameフィールドのエラーが発生するべきです (atname=%s)", atname)
		}
	}
}

func TestAccountsCreateValidator_AtnameTooLong(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, AccountsCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "a23456789012345678901", // 21文字
		Password: "password123",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if !result.FormErrors.HasErrors() {
		t.Error("バリデーションエラーが発生するべきです")
	}

	atnameErrors := result.FormErrors.Fields["atname"]
	if len(atnameErrors) == 0 {
		t.Error("atnameフィールドのエラーが発生するべきです")
	}
}

func TestAccountsCreateValidator_ReservedAtname(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	reservedNames := []string{
		"admin",
		"Admin",
		"ADMIN",
		"support",
		"mewst",
	}

	for _, atname := range reservedNames {
		result := validator.Validate(ctx, AccountsCreateValidatorInput{
			Email:    "test@example.com",
			Atname:   atname,
			Password: "password123",
		})

		if result.Err != nil {
			t.Fatalf("予期しないエラー (atname=%s): %v", atname, result.Err)
		}

		if !result.FormErrors.HasErrors() {
			t.Errorf("バリデーションエラーが発生するべきです (atname=%s)", atname)
		}

		atnameErrors := result.FormErrors.Fields["atname"]
		if len(atnameErrors) == 0 {
			t.Errorf("atnameフィールドのエラーが発生するべきです (atname=%s)", atname)
		}
	}
}

func TestAccountsCreateValidator_AtnameAlreadyTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	// 既存のプロフィールを作成
	testutil.NewProfileBuilder(t, tx).
		WithAtname("existinguser").
		Build()

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, AccountsCreateValidatorInput{
		Email:    "newuser@example.com",
		Atname:   "existinguser",
		Password: "password123",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if !result.FormErrors.HasErrors() {
		t.Error("バリデーションエラーが発生するべきです")
	}

	atnameErrors := result.FormErrors.Fields["atname"]
	if len(atnameErrors) == 0 {
		t.Error("atnameフィールドのエラーが発生するべきです")
	}
}

func TestAccountsCreateValidator_EmailAlreadyTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	// 既存のユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("existing@example.com").
		WithPasswordDigest("$2a$10$test").
		Build()

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, AccountsCreateValidatorInput{
		Email:    "existing@example.com",
		Atname:   "newuser",
		Password: "password123",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if !result.FormErrors.HasErrors() {
		t.Error("バリデーションエラーが発生するべきです")
	}

	// メールアドレスは編集不可のため、グローバルエラーとして表示される
	if len(result.FormErrors.Global) == 0 {
		t.Error("グローバルエラーが発生するべきです")
	}
}

func TestAccountsCreateValidator_EmptyPassword(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, AccountsCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "testuser",
		Password: "",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if !result.FormErrors.HasErrors() {
		t.Error("バリデーションエラーが発生するべきです")
	}

	passwordErrors := result.FormErrors.Fields["password"]
	if len(passwordErrors) == 0 {
		t.Error("passwordフィールドのエラーが発生するべきです")
	}
}

func TestAccountsCreateValidator_PasswordTooShort(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, AccountsCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "testuser",
		Password: "short",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if !result.FormErrors.HasErrors() {
		t.Error("バリデーションエラーが発生するべきです")
	}

	passwordErrors := result.FormErrors.Fields["password"]
	if len(passwordErrors) == 0 {
		t.Error("passwordフィールドのエラーが発生するべきです")
	}
}

func TestAccountsCreateValidator_PasswordTooLong(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	// 73文字のパスワード（bcryptの72文字制限を超える）
	longPassword := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"

	result := validator.Validate(ctx, AccountsCreateValidatorInput{
		Email:    "test@example.com",
		Atname:   "testuser",
		Password: longPassword,
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if !result.FormErrors.HasErrors() {
		t.Error("バリデーションエラーが発生するべきです")
	}

	passwordErrors := result.FormErrors.Fields["password"]
	if len(passwordErrors) == 0 {
		t.Error("passwordフィールドのエラーが発生するべきです")
	}
}

func TestAccountsCreateValidator_ValidInput(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	result := validator.Validate(ctx, AccountsCreateValidatorInput{
		Email:    "newuser@example.com",
		Atname:   "newuser",
		Password: "password123",
	})

	if result.Err != nil {
		t.Fatalf("予期しないエラー: %v", result.Err)
	}

	if result.FormErrors.HasErrors() {
		t.Errorf("バリデーションエラーが発生すべきではありません: %v", result.FormErrors.Fields)
	}
}

func TestAccountsCreateValidator_ValidAtnameFormats(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	validator := NewAccountsCreateValidator(userRepo, profileRepo)

	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")

	validAtnames := []string{
		"user123",
		"User_Name",
		"test",
		"a",
		"12345678901234567890", // ちょうど20文字
	}

	for _, atname := range validAtnames {
		result := validator.Validate(ctx, AccountsCreateValidatorInput{
			Email:    "test@example.com",
			Atname:   atname,
			Password: "password123",
		})

		if result.Err != nil {
			t.Fatalf("予期しないエラー (atname=%s): %v", atname, result.Err)
		}

		if result.FormErrors.HasErrors() {
			t.Errorf("バリデーションエラーが発生すべきではありません (atname=%s): %v", atname, result.FormErrors.Fields)
		}
	}
}
