package validator

import (
	"context"
	"net/url"
	"strings"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
)

// LinkDataFetcherValidator validates the link card target URL, mirroring the
// Rails LinkDataFetcherForm (presence + URL format).
//
// [Ja] LinkDataFetcherValidator はリンクカードの対象 URL をバリデーションする。
// Rails の LinkDataFetcherForm (presence + URL 形式) に対応する。
type LinkDataFetcherValidator struct{}

// NewLinkDataFetcherValidator は LinkDataFetcherValidator を生成する
func NewLinkDataFetcherValidator() *LinkDataFetcherValidator {
	return &LinkDataFetcherValidator{}
}

// LinkDataFetcherValidatorInput はバリデーションの入力パラメータ
type LinkDataFetcherValidatorInput struct {
	TargetURL string
}

// Validate checks the format of the target URL (no DB access).
// [Ja] Validate は対象 URL の形式をチェックする (DB アクセスなし)。
func (v *LinkDataFetcherValidator) Validate(ctx context.Context, input LinkDataFetcherValidatorInput) error {
	ve := model.NewValidationError()

	// Treat a whitespace-only value as blank, matching Rails' presence: true
	// (blank?) just like the post content validation.
	// [Ja] 空白のみの値も blank とみなし、投稿本文のバリデーションと同様に Rails の
	// presence: true (blank?) に揃える。
	if strings.TrimSpace(input.TargetURL) == "" {
		ve.AddField("target_url", i18n.T(ctx, "validation_required"))
		return ve
	}

	if !IsValidURL(input.TargetURL) {
		ve.AddField("target_url", i18n.T(ctx, "validation_url_invalid"))
		return ve
	}

	return nil
}

// IsValidURL reports whether the value is a parsable URL with a host,
// mirroring the Rails Url#valid? (parsable by Addressable + host present).
// It is exported so the link metadata fetch UseCase can apply the same check
// to the fetched canonical / image URLs (Rails LinkForm's url validator).
//
// [Ja] IsValidURL は値がホストを持つパース可能な URL かどうかを返す。Rails の
// Url#valid? (Addressable でパース可能 + host あり) に対応する。リンクメタデータ
// 取得 UseCase が取得した canonical / image URL に同じチェック (Rails LinkForm の
// url バリデータ) を適用できるよう公開している。
func IsValidURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Host != ""
}
