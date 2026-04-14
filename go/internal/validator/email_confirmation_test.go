package validator

import (
	"context"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestEmailConfirmationCreateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	validator := NewEmailConfirmationCreateValidator(emailConfirmationRepo)

	tests := []struct {
		name          string
		code          string
		wantErrors    bool
		expectedField string
	}{
		{
			name:          "異常系: コードが空",
			code:          "",
			wantErrors:    true,
			expectedField: "code",
		},
		{
			name:          "異常系: 5桁のコード（短すぎる）",
			code:          "12345",
			wantErrors:    true,
			expectedField: "code",
		},
		{
			name:          "異常系: 7桁のコード（長すぎる）",
			code:          "1234567",
			wantErrors:    true,
			expectedField: "code",
		},
		{
			name:          "異常系: アルファベットを含むコード",
			code:          "12345a",
			wantErrors:    true,
			expectedField: "code",
		},
		{
			name:          "異常系: 記号を含むコード",
			code:          "12345-",
			wantErrors:    true,
			expectedField: "code",
		},
		{
			name:          "異常系: スペースを含むコード",
			code:          "123 456",
			wantErrors:    true,
			expectedField: "code",
		},
		{
			name:          "異常系: 全角数字を含むコード",
			code:          "１２３４５６",
			wantErrors:    true,
			expectedField: "code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := EmailConfirmationCreateValidatorInput{
				ID:   testutil.MustParseUUID("00000000-0000-0000-0000-000000000000"),
				Code: tt.code,
			}

			_, err := validator.Validate(ctx, input)

			if tt.wantErrors {
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Error("エラーが期待されたが、エラーがありません")
				}
				if tt.expectedField != "" && !ve.HasFieldError(tt.expectedField) {
					t.Errorf("フィールド %q のエラーが期待されましたが、ありません", tt.expectedField)
				}
			}
		})
	}
}

func TestEmailConfirmationCreateValidator_Validate_ValidCodeFormats(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	validator := NewEmailConfirmationCreateValidator(emailConfirmationRepo)

	tests := []struct {
		name string
		code string
	}{
		{
			name: "有効な6桁コード",
			code: "123456",
		},
		{
			name: "先頭が0の6桁コード",
			code: "012345",
		},
		{
			name: "すべて0の6桁コード",
			code: "000000",
		},
		{
			name: "すべて9の6桁コード",
			code: "999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := EmailConfirmationCreateValidatorInput{
				ID:   testutil.MustParseUUID("00000000-0000-0000-0000-000000000000"),
				Code: tt.code,
			}

			_, err := validator.Validate(ctx, input)

			// 形式バリデーションでエラーにならないことを確認
			// （レコードが存在しないためグローバルエラーは発生するが、フィールドエラーは発生しない）
			ve := model.AsValidationError(err)
			if ve != nil && ve.HasFieldError("code") {
				t.Errorf("code形式バリデーションでエラーが発生: %v", ve.GetFieldErrors("code"))
			}
		})
	}
}

func TestCodeRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code  string
		valid bool
	}{
		{"123456", true},
		{"000000", true},
		{"999999", true},
		{"12345", false},
		{"1234567", false},
		{"abcdef", false},
		{"12345a", false},
		{"", false},
		{"123 456", false},
		{"12-345", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			t.Parallel()

			if got := codeRegex.MatchString(tt.code); got != tt.valid {
				t.Errorf("codeRegex.MatchString(%q) = %v, want %v", tt.code, got, tt.valid)
			}
		})
	}
}

func TestEmailConfirmationCreateValidator_Validate_Success(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// テスト用のメール確認レコードを作成
	id := testutil.NewEmailConfirmationBuilder(t, tx).
		WithCode("123456").
		Build()

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	validator := NewEmailConfirmationCreateValidator(emailConfirmationRepo)

	input := EmailConfirmationCreateValidatorInput{
		ID:   id,
		Code: "123456",
	}

	output, err := validator.Validate(ctx, input)

	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if output == nil {
		t.Fatal("Validate() output = nil, want non-nil")
	}
	if output.EmailConfirmation == nil {
		t.Error("Validate() emailConfirmation = nil, want non-nil")
	}
	if output.EmailConfirmation != nil && output.EmailConfirmation.Email == "" {
		t.Error("Validate() emailConfirmation.Email should not be empty")
	}
}

func TestEmailConfirmationCreateValidator_Validate_RecordNotFound(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// レコードを作成しない

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	validator := NewEmailConfirmationCreateValidator(emailConfirmationRepo)

	// 存在しないIDを使用
	input := EmailConfirmationCreateValidatorInput{
		ID:   testutil.MustParseUUID("00000000-0000-0000-0000-000000000000"),
		Code: "123456",
	}

	output, err := validator.Validate(ctx, input)

	if output != nil {
		t.Error("Validate() output should be nil for non-existent record")
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("Validate() expected ValidationError")
	}
	if !ve.HasErrors() {
		t.Error("Validate() formErrors should have errors")
	}
}

func TestEmailConfirmationCreateValidator_Validate_ExpiredRecord(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// 期限切れのレコードを作成（31分前に作成）
	expiredTime := time.Now().Add(-31 * time.Minute)
	id := testutil.NewEmailConfirmationBuilder(t, tx).
		WithCode("123456").
		WithCreatedAt(expiredTime).
		Build()

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	validator := NewEmailConfirmationCreateValidator(emailConfirmationRepo)

	input := EmailConfirmationCreateValidatorInput{
		ID:   id,
		Code: "123456",
	}

	output, err := validator.Validate(ctx, input)

	if output != nil {
		t.Error("Validate() output should be nil for expired record")
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("Validate() expected ValidationError")
	}
	if !ve.HasErrors() {
		t.Error("Validate() formErrors should have errors")
	}
}

func TestEmailConfirmationCreateValidator_Validate_AlreadySucceeded(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// 既に成功済みのレコードを作成
	succeededAt := time.Now().Add(-10 * time.Minute)
	id := testutil.NewEmailConfirmationBuilder(t, tx).
		WithCode("123456").
		WithSucceededAt(succeededAt).
		Build()

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	validator := NewEmailConfirmationCreateValidator(emailConfirmationRepo)

	input := EmailConfirmationCreateValidatorInput{
		ID:   id,
		Code: "123456",
	}

	output, err := validator.Validate(ctx, input)

	if output != nil {
		t.Error("Validate() output should be nil for already succeeded record")
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("Validate() expected ValidationError")
	}
	if !ve.HasErrors() {
		t.Error("Validate() formErrors should have errors")
	}
}

func TestEmailConfirmationCreateValidator_Validate_InvalidCode(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// テスト用のメール確認レコードを作成
	id := testutil.NewEmailConfirmationBuilder(t, tx).
		WithCode("123456").
		Build()

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	validator := NewEmailConfirmationCreateValidator(emailConfirmationRepo)

	input := EmailConfirmationCreateValidatorInput{
		ID:   id,
		Code: "654321", // 異なるコード
	}

	output, err := validator.Validate(ctx, input)

	if output != nil {
		t.Error("Validate() output should be nil for invalid code")
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("Validate() expected ValidationError")
	}
	if !ve.HasErrors() {
		t.Error("Validate() formErrors should have errors")
	}
}

func TestEmailConfirmationCreateValidator_Validate_GlobalError(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// テスト用のメール確認レコードを作成
	id := testutil.NewEmailConfirmationBuilder(t, tx).
		WithCode("123456").
		Build()

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	validator := NewEmailConfirmationCreateValidator(emailConfirmationRepo)

	input := EmailConfirmationCreateValidatorInput{
		ID:   id,
		Code: "654321", // 異なるコード
	}

	_, err := validator.Validate(ctx, input)

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError")
	}

	// グローバルエラーとして返されることを確認（フィールドエラーではない）
	if len(ve.Global) == 0 {
		t.Error("expected global error, not field error")
	}
	if len(ve.Fields) > 0 {
		t.Error("should not have field errors for code validation")
	}
}

func TestEmailConfirmationCreateValidator_Validate_ErrorMessageIsGeneric(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// テスト用のメール確認レコードを作成
	id := testutil.NewEmailConfirmationBuilder(t, tx).
		WithCode("123456").
		Build()

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	validator := NewEmailConfirmationCreateValidator(emailConfirmationRepo)

	t.Run("コードが一致しない場合も期限切れと同じエラーメッセージ", func(t *testing.T) {
		t.Parallel()

		// コードが一致しない場合
		input := EmailConfirmationCreateValidatorInput{
			ID:   id,
			Code: "654321",
		}

		_, err := validator.Validate(ctx, input)

		ve := model.AsValidationError(err)
		if ve == nil || len(ve.Global) == 0 {
			t.Fatal("expected global error message")
		}

		invalidCodeMsg := ve.Global[0]

		// 存在しないIDの場合
		input2 := EmailConfirmationCreateValidatorInput{
			ID:   testutil.MustParseUUID("00000000-0000-0000-0000-000000000001"),
			Code: "123456",
		}

		_, err2 := validator.Validate(ctx, input2)

		ve2 := model.AsValidationError(err2)
		if ve2 == nil || len(ve2.Global) == 0 {
			t.Fatal("expected global error message")
		}

		notFoundMsg := ve2.Global[0]

		// セキュリティ上、両方のエラーメッセージが同じであることを確認
		if invalidCodeMsg != notFoundMsg {
			t.Errorf("エラーメッセージが異なります: invalid code = %q, not found = %q", invalidCodeMsg, notFoundMsg)
		}
	})
}
