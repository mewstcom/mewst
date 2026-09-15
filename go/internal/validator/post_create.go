package validator

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
)

// PostCreateValidatorは投稿作成フォームのバリデーションを行う
type PostCreateValidator struct{}

// NewPostCreateValidatorはPostCreateValidatorを生成する
func NewPostCreateValidator() *PostCreateValidator {
	return &PostCreateValidator{}
}

// PostCreateValidatorInputはバリデーションの入力パラメータ
type PostCreateValidatorInput struct {
	Content string
}

// Validateは投稿本文の形式をチェックする (DBアクセスなし)。
func (v *PostCreateValidator) Validate(ctx context.Context, input PostCreateValidatorInput) error {
	ve := model.NewValidationError()

	// 空文字列だけでなく空白のみの本文もblankとみなし、Railsのpresence: true
	// (blank?) に揃える。トリムしないと空白のみ投稿がGoでは通過しRailsでは弾かれ、
	// 共有DB上で挙動が食い違ってしまう。
	if strings.TrimSpace(input.Content) == "" {
		ve.AddField("content", i18n.T(ctx, "validation_required"))
		return ve
	}

	// マルチバイト文字 (日本語など) を1文字として数えるためrune単位で計測し、
	// Railsの文字数ベースのlengthバリデーションに揃える。
	if utf8.RuneCountInString(input.Content) > model.MaximumPostContentLength {
		ve.AddField("content", i18n.T(ctx, "validation_content_too_long"))
		return ve
	}

	return nil
}
