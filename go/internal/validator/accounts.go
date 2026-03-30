package validator

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// atnameRegex はアットネームの形式チェック用正規表現
var atnameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// reservedAtnames は予約済みのアットネーム一覧
var reservedAtnames = map[string]bool{
	"admin":         true,
	"administrator": true,
	"support":       true,
	"help":          true,
	"info":          true,
	"contact":       true,
	"sales":         true,
	"marketing":     true,
	"noreply":       true,
	"postmaster":    true,
	"webmaster":     true,
	"root":          true,
	"system":        true,
	"api":           true,
	"mewst":         true,
	"official":      true,
	"news":          true,
	"blog":          true,
	"status":        true,
}

// maxAtnameLength はアットネームの最大文字数
const maxAtnameLength = 20

// minPasswordLength はパスワードの最小文字数
const minPasswordLength = 8

// maxPasswordLength はパスワードの最大文字数（bcryptの制限）
const maxPasswordLength = 72

// AccountsCreateValidator はアカウント作成フォームのバリデーションを行う
type AccountsCreateValidator struct {
	userRepo    *repository.UserRepository
	profileRepo *repository.ProfileRepository
}

// NewAccountsCreateValidator はAccountsCreateValidatorを生成する
func NewAccountsCreateValidator(userRepo *repository.UserRepository, profileRepo *repository.ProfileRepository) *AccountsCreateValidator {
	return &AccountsCreateValidator{
		userRepo:    userRepo,
		profileRepo: profileRepo,
	}
}

// AccountsCreateValidatorInput はバリデーションの入力パラメータ
type AccountsCreateValidatorInput struct {
	Email    string
	Atname   string
	Password string
}

// AccountsCreateValidatorResult はバリデーションの結果
type AccountsCreateValidatorResult struct {
	FormErrors *session.FormErrors
	Err        error
}

// Validate は入力値をチェックする（形式チェック + DB検証）
func (v *AccountsCreateValidator) Validate(ctx context.Context, input AccountsCreateValidatorInput) *AccountsCreateValidatorResult {
	formErrors := session.NewFormErrors()

	// アットネームのバリデーション
	v.validateAtname(ctx, formErrors, input.Atname)

	// パスワードのバリデーション
	v.validatePassword(ctx, formErrors, input.Password)

	// 形式バリデーションでエラーがあれば早期リターン
	if formErrors.HasErrors() {
		return &AccountsCreateValidatorResult{FormErrors: formErrors}
	}

	// 状態バリデーション（DB検証）
	if err := v.validateAtnameUniqueness(ctx, formErrors, input.Atname); err != nil {
		return &AccountsCreateValidatorResult{Err: err}
	}

	if err := v.validateEmailUniqueness(ctx, formErrors, input.Email); err != nil {
		return &AccountsCreateValidatorResult{Err: err}
	}

	return &AccountsCreateValidatorResult{FormErrors: formErrors}
}

// validateAtname はアットネームの形式バリデーションを行う
func (v *AccountsCreateValidator) validateAtname(ctx context.Context, formErrors *session.FormErrors, atname string) {
	if atname == "" {
		formErrors.AddFieldError("atname", templates.T(ctx, "error_required"))
		return
	}

	if !atnameRegex.MatchString(atname) {
		formErrors.AddFieldError("atname", templates.T(ctx, "error_atname_format"))
		return
	}

	if len(atname) > maxAtnameLength {
		formErrors.AddFieldError("atname", templates.T(ctx, "error_atname_too_long"))
		return
	}

	if reservedAtnames[strings.ToLower(atname)] {
		formErrors.AddFieldError("atname", templates.T(ctx, "error_atname_reserved"))
		return
	}
}

// validatePassword はパスワードの形式バリデーションを行う
func (v *AccountsCreateValidator) validatePassword(ctx context.Context, formErrors *session.FormErrors, password string) {
	if password == "" {
		formErrors.AddFieldError("password", templates.T(ctx, "error_required"))
		return
	}

	if utf8.RuneCountInString(password) < minPasswordLength {
		formErrors.AddFieldError("password", templates.T(ctx, "error_password_too_short"))
		return
	}

	if len(password) > maxPasswordLength {
		formErrors.AddFieldError("password", templates.T(ctx, "error_password_too_long"))
		return
	}
}

// validateAtnameUniqueness はアットネームの重複チェックを行う
func (v *AccountsCreateValidator) validateAtnameUniqueness(ctx context.Context, formErrors *session.FormErrors, atname string) error {
	exists, err := v.profileRepo.ExistsByAtname(ctx, atname)
	if err != nil {
		return err
	}
	if exists {
		formErrors.AddFieldError("atname", templates.T(ctx, "error_atname_already_taken"))
	}
	return nil
}

// validateEmailUniqueness はメールアドレスの重複チェックを行う
// アカウント作成フォームではメールアドレスは編集不可のため、グローバルエラーとして表示する
func (v *AccountsCreateValidator) validateEmailUniqueness(ctx context.Context, formErrors *session.FormErrors, email string) error {
	exists, err := v.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		formErrors.AddGlobalError(templates.T(ctx, "error_email_already_taken"))
	}
	return nil
}
