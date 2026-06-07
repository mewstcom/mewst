package validator

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
)

// PostCreateValidator は投稿作成フォームのバリデーションを行う
type PostCreateValidator struct{}

// NewPostCreateValidator は PostCreateValidator を生成する
func NewPostCreateValidator() *PostCreateValidator {
	return &PostCreateValidator{}
}

// PostCreateValidatorInput はバリデーションの入力パラメータ
type PostCreateValidatorInput struct {
	Content string
}

// Validate checks the format of the post body (no DB access).
// [Ja] Validate は投稿本文の形式をチェックする (DB アクセスなし)。
func (v *PostCreateValidator) Validate(ctx context.Context, input PostCreateValidatorInput) error {
	ve := model.NewValidationError()

	// Treat an empty or whitespace-only body as blank, matching Rails'
	// presence: true (blank?). Without trimming, a whitespace-only post would
	// pass here in Go but be rejected by Rails, diverging on the shared DB.
	//
	// [Ja] 空文字列だけでなく空白のみの本文も blank とみなし、Rails の presence: true
	// (blank?) に揃える。トリムしないと空白のみ投稿が Go では通過し Rails では弾かれ、
	// 共有 DB 上で挙動が食い違ってしまう。
	if strings.TrimSpace(input.Content) == "" {
		ve.AddField("content", i18n.T(ctx, "validation_required"))
		return ve
	}

	// Count by rune so that multibyte characters (e.g. Japanese) are counted as
	// one each, matching Rails' character-based length validation.
	// [Ja] マルチバイト文字 (日本語など) を 1 文字として数えるため rune 単位で計測し、
	// Rails の文字数ベースの length バリデーションに揃える。
	if utf8.RuneCountInString(input.Content) > model.MaximumPostContentLength {
		ve.AddField("content", i18n.T(ctx, "validation_content_too_long"))
		return ve
	}

	return nil
}
