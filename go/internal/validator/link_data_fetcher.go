package validator

import (
	"context"
	"net/url"
	"strings"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
)

// LinkDataFetcherValidatorはリンクカードの対象URLをバリデーションする。
// RailsのLinkDataFetcherForm (presence + URL形式) に対応する。
type LinkDataFetcherValidator struct{}

// NewLinkDataFetcherValidatorはLinkDataFetcherValidatorを生成する
func NewLinkDataFetcherValidator() *LinkDataFetcherValidator {
	return &LinkDataFetcherValidator{}
}

// LinkDataFetcherValidatorInputはバリデーションの入力パラメータ
type LinkDataFetcherValidatorInput struct {
	TargetURL string
}

// Validateは対象URLの形式をチェックする (DBアクセスなし)。
func (v *LinkDataFetcherValidator) Validate(ctx context.Context, input LinkDataFetcherValidatorInput) error {
	ve := model.NewValidationError()

	// 空白のみの値もblankとみなし、投稿本文のバリデーションと同様にRailsの
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

// IsValidURLは値がホストを持つパース可能なURLかどうかを返す。Railsの
// Url#valid? (Addressableでパース可能 + hostあり) に対応する。リンクメタデータ
// 取得UseCaseが取得したcanonical / image URLに同じチェック (Rails LinkFormの
// urlバリデータ) を適用できるよう公開している。
func IsValidURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Host != ""
}
