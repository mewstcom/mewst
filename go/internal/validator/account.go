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

// atnameRegexはアットネームの形式チェック用正規表現
var atnameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// reservedAtnamesは予約済みのアットネーム一覧
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

// AtnameMaxLengthはアットネームの最大文字数。
const AtnameMaxLength = 20

// minPasswordLengthはパスワードの最小文字数
const minPasswordLength = 8

// PasswordMaxBytesはbcryptがハッシュ化できるパスワードの最大バイト数。
// アカウント作成フォームの外でアカウントを書き込む呼び出し元が、制約の写しでは
// なく同じ上限を検査できるよう公開している。
const PasswordMaxBytes = 72

// IsValidAtnameは、アットネームがすべてのアカウントに課される形式と長さを
// 満たすかを返す。アカウント作成フォームの外でアカウントを作る呼び出し元が、制約の
// 写しではなく同じ規則を参照できるよう公開している。
//
// AccountCreateValidatorは同じ2つの制約を個別に適用する。フォームは2つを別の
// メッセージで伝え分けるため。ここへ規則を足したときは、あちらへも足す必要がある。
// そうしないと、フォームが受理するatnameを名簿が拒否することになる。
func IsValidAtname(atname string) bool {
	return len(atname) <= AtnameMaxLength && atnameRegex.MatchString(atname)
}

// AccountCreateValidatorはアカウント作成フォームのバリデーションを行う
type AccountCreateValidator struct {
	userRepo    *repository.UserRepository
	profileRepo *repository.ProfileRepository
}

// NewAccountCreateValidatorはAccountCreateValidatorを生成する
func NewAccountCreateValidator(userRepo *repository.UserRepository, profileRepo *repository.ProfileRepository) *AccountCreateValidator {
	return &AccountCreateValidator{
		userRepo:    userRepo,
		profileRepo: profileRepo,
	}
}

// AccountCreateValidatorInputはバリデーションの入力パラメータ
type AccountCreateValidatorInput struct {
	Email    string
	Atname   string
	Password string
}

// Validateは入力値をチェックする (形式チェック + DB検証)
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

	// 状態バリデーション (DB検証)
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

// validateAtnameはアットネームの形式バリデーションを行う
func (v *AccountCreateValidator) validateAtname(ctx context.Context, ve *model.ValidationError, atname string) {
	if atname == "" {
		ve.AddField("atname", i18n.T(ctx, "validation_required"))
		return
	}

	if !atnameRegex.MatchString(atname) {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_invalid_format"))
		return
	}

	if len(atname) > AtnameMaxLength {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_too_long"))
		return
	}

	if reservedAtnames[strings.ToLower(atname)] {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_reserved"))
		return
	}
}

// validatePasswordはパスワードの形式バリデーションを行う
func (v *AccountCreateValidator) validatePassword(ctx context.Context, ve *model.ValidationError, password string) {
	if password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_required"))
		return
	}

	if utf8.RuneCountInString(password) < minPasswordLength {
		ve.AddField("password", i18n.T(ctx, "validation_password_too_short"))
		return
	}

	if len(password) > PasswordMaxBytes {
		ve.AddField("password", i18n.T(ctx, "validation_password_too_long"))
		return
	}
}

// validateAtnameUniquenessはアットネームの重複チェックを行う
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

// validateEmailUniquenessはメールアドレスの重複チェックを行う
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
