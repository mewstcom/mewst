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

// AccountCreateValidator はアカウント作成フォームのバリデーションを行う
type AccountCreateValidator struct {
	userRepo    *repository.UserRepository
	profileRepo *repository.ProfileRepository
}

// NewAccountCreateValidator はAccountCreateValidatorを生成する
func NewAccountCreateValidator(userRepo *repository.UserRepository, profileRepo *repository.ProfileRepository) *AccountCreateValidator {
	return &AccountCreateValidator{
		userRepo:    userRepo,
		profileRepo: profileRepo,
	}
}

// AccountCreateValidatorInput はバリデーションの入力パラメータ
type AccountCreateValidatorInput struct {
	Email    string
	Atname   string
	Password string
}

// Validate は入力値をチェックする（形式チェック + DB検証）
func (v *AccountCreateValidator) Validate(ctx context.Context, input AccountCreateValidatorInput) error {
	ve := model.NewValidationError()

	// アットネームのバリデーション
	v.validateAtname(ctx, ve, input.Atname)

	// パスワードのバリデーション
	v.validatePassword(ctx, ve, input.Password)

	// 形式バリデーションでエラーがあれば早期リターン
	if ve.HasErrors() {
		return ve
	}

	// 状態バリデーション（DB検証）
	if err := v.validateAtnameUniqueness(ctx, ve, input.Atname); err != nil {
		return err
	}

	if err := v.validateEmailUniqueness(ctx, ve, input.Email); err != nil {
		return err
	}

	if ve.HasErrors() {
		return ve
	}

	return nil
}

// validateAtname はアットネームの形式バリデーションを行う
func (v *AccountCreateValidator) validateAtname(ctx context.Context, ve *model.ValidationError, atname string) {
	if atname == "" {
		ve.AddField("atname", i18n.T(ctx, "validation_required"))
		return
	}

	if !atnameRegex.MatchString(atname) {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_invalid_format"))
		return
	}

	if len(atname) > maxAtnameLength {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_too_long"))
		return
	}

	if reservedAtnames[strings.ToLower(atname)] {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_reserved"))
		return
	}
}

// validatePassword はパスワードの形式バリデーションを行う
func (v *AccountCreateValidator) validatePassword(ctx context.Context, ve *model.ValidationError, password string) {
	if password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_required"))
		return
	}

	if utf8.RuneCountInString(password) < minPasswordLength {
		ve.AddField("password", i18n.T(ctx, "validation_password_too_short"))
		return
	}

	if len(password) > maxPasswordLength {
		ve.AddField("password", i18n.T(ctx, "validation_password_too_long"))
		return
	}
}

// validateAtnameUniqueness はアットネームの重複チェックを行う
func (v *AccountCreateValidator) validateAtnameUniqueness(ctx context.Context, ve *model.ValidationError, atname string) error {
	exists, err := v.profileRepo.ExistsByAtname(ctx, atname)
	if err != nil {
		return err
	}
	if exists {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_already_taken"))
	}
	return nil
}

// validateEmailUniqueness はメールアドレスの重複チェックを行う
// アカウント作成フォームではメールアドレスは編集不可のため、グローバルエラーとして表示する
func (v *AccountCreateValidator) validateEmailUniqueness(ctx context.Context, ve *model.ValidationError, email string) error {
	exists, err := v.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		ve.AddGlobal(i18n.T(ctx, "validation_email_already_taken"))
	}
	return nil
}
