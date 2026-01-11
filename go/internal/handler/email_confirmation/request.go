package email_confirmation

import (
	"context"
	"regexp"

	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
)

// 確認コードは6桁の数字
var codeRegex = regexp.MustCompile(`^\d{6}$`)

// CreateRequest は確認コード入力フォームのリクエストデータ
type CreateRequest struct {
	Code string
}

// Validate はリクエストデータを検証する
func (r *CreateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// 確認コードの必須チェック
	if r.Code == "" {
		errors.AddFieldError("code", templates.T(ctx, "error_required"))
	} else {
		// 6桁の数字形式チェック
		if !codeRegex.MatchString(r.Code) {
			errors.AddFieldError("code", templates.T(ctx, "error_invalid_code_format"))
		}
	}

	return errors
}
