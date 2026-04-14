package validator

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
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

// AccountsCreateValidatorOutput はバリデーション成功時の出力
type AccountsCreateValidatorOutput struct{}

// Validate は入力値をチェックする（形式チェック + DB検証）
func (v *AccountsCreateValidator) Validate(ctx context.Context, input AccountsCreateValidatorInput) (*AccountsCreateValidatorOutput, error) {
	ve := model.NewValidationError()

	// アットネームのバリデーション
	v.validateAtname(ctx, ve, input.Atname)

	// パスワードのバリデーション
	v.validatePassword(ctx, ve, input.Password)

	// 形式バリデーションでエラーがあれば早期リターン
	if ve.HasErrors() {
		return nil, ve
	}

	// 状態バリデーション（DB検証）
	if err := v.validateAtnameUniqueness(ctx, ve, input.Atname); err != nil {
		return nil, err
	}

	if err := v.validateEmailUniqueness(ctx, ve, input.Email); err != nil {
		return nil, err
	}

	if ve.HasErrors() {
		return nil, ve
	}

	return &AccountsCreateValidatorOutput{}, nil
}

// validateAtname はアットネームの形式バリデーションを行う
func (v *AccountsCreateValidator) validateAtname(ctx context.Context, ve *model.ValidationError, atname string) {
	if atname == "" {
		ve.AddField("atname", i18n.T(ctx, "error_required"))
		return
	}

	if !atnameRegex.MatchString(atname) {
		ve.AddField("atname", i18n.T(ctx, "error_atname_format"))
		return
	}

	if len(atname) > maxAtnameLength {
		ve.AddField("atname", i18n.T(ctx, "error_atname_too_long"))
		return
	}

	if reservedAtnames[strings.ToLower(atname)] {
		ve.AddField("atname", i18n.T(ctx, "error_atname_reserved"))
		return
	}
}

// validatePassword はパスワードの形式バリデーションを行う
func (v *AccountsCreateValidator) validatePassword(ctx context.Context, ve *model.ValidationError, password string) {
	if password == "" {
		ve.AddField("password", i18n.T(ctx, "error_required"))
		return
	}

	if utf8.RuneCountInString(password) < minPasswordLength {
		ve.AddField("password", i18n.T(ctx, "error_password_too_short"))
		return
	}

	if len(password) > maxPasswordLength {
		ve.AddField("password", i18n.T(ctx, "error_password_too_long"))
		return
	}
}

// validateAtnameUniqueness はアットネームの重複チェックを行う
func (v *AccountsCreateValidator) validateAtnameUniqueness(ctx context.Context, ve *model.ValidationError, atname string) error {
	exists, err := v.profileRepo.ExistsByAtname(ctx, atname)
	if err != nil {
		return err
	}
	if exists {
		ve.AddField("atname", i18n.T(ctx, "error_atname_already_taken"))
	}
	return nil
}

// validateEmailUniqueness はメールアドレスの重複チェックを行う
// アカウント作成フォームではメールアドレスは編集不可のため、グローバルエラーとして表示する
func (v *AccountsCreateValidator) validateEmailUniqueness(ctx context.Context, ve *model.ValidationError, email string) error {
	exists, err := v.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		ve.AddGlobal(i18n.T(ctx, "error_email_already_taken"))
	}
	return nil
}
