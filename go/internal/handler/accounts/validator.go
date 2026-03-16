package accounts

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

// CreateValidator はアカウント作成フォームのバリデーションを行う
type CreateValidator struct {
	userRepo    *repository.UserRepository
	profileRepo *repository.ProfileRepository
}

// NewCreateValidator はCreateValidatorを生成する
func NewCreateValidator(userRepo *repository.UserRepository, profileRepo *repository.ProfileRepository) *CreateValidator {
	return &CreateValidator{
		userRepo:    userRepo,
		profileRepo: profileRepo,
	}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	Email    string
	Atname   string
	Password string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	FormErrors *session.FormErrors
	Err        error
}

// Validate は入力値をチェックする（形式チェック + DB検証）
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
	formErrors := session.NewFormErrors()

	// アットネームのバリデーション
	v.validateAtname(ctx, formErrors, input.Atname)

	// パスワードのバリデーション
	v.validatePassword(ctx, formErrors, input.Password)

	// 形式バリデーションでエラーがあれば早期リターン
	if formErrors.HasErrors() {
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	// 状態バリデーション（DB検証）
	if err := v.validateAtnameUniqueness(ctx, formErrors, input.Atname); err != nil {
		return &CreateValidatorResult{Err: err}
	}

	if err := v.validateEmailUniqueness(ctx, formErrors, input.Email); err != nil {
		return &CreateValidatorResult{Err: err}
	}

	return &CreateValidatorResult{FormErrors: formErrors}
}

// validateAtname はアットネームの形式バリデーションを行う
func (v *CreateValidator) validateAtname(ctx context.Context, formErrors *session.FormErrors, atname string) {
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
func (v *CreateValidator) validatePassword(ctx context.Context, formErrors *session.FormErrors, password string) {
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
func (v *CreateValidator) validateAtnameUniqueness(ctx context.Context, formErrors *session.FormErrors, atname string) error {
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
func (v *CreateValidator) validateEmailUniqueness(ctx context.Context, formErrors *session.FormErrors, email string) error {
	exists, err := v.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		formErrors.AddGlobalError(templates.T(ctx, "error_email_already_taken"))
	}
	return nil
}
