package validator

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
)

func TestLinkDataFetcherValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         LinkDataFetcherValidatorInput
		wantErrors    bool
		expectedField string
	}{
		{
			name:       "正常系: http の URL",
			input:      LinkDataFetcherValidatorInput{TargetURL: "http://example.com"},
			wantErrors: false,
		},
		{
			name:       "正常系: https の URL (パス・クエリ付き)",
			input:      LinkDataFetcherValidatorInput{TargetURL: "https://example.com/articles/1?ref=home"},
			wantErrors: false,
		},
		{
			name:          "異常系: URL が空",
			input:         LinkDataFetcherValidatorInput{TargetURL: ""},
			wantErrors:    true,
			expectedField: "target_url",
		},
		{
			name:          "異常系: 空白のみ",
			input:         LinkDataFetcherValidatorInput{TargetURL: "   "},
			wantErrors:    true,
			expectedField: "target_url",
		},
		{
			// A host-less value like a bare domain is not a valid URL in Rails
			// (Url#valid? requires a host) either.
			// [Ja] スキームの無い素のドメインはホストを持たないため、Rails の Url#valid?
			// (host 必須) でも無効になる。
			name:          "異常系: スキーム無しのドメイン",
			input:         LinkDataFetcherValidatorInput{TargetURL: "example.com"},
			wantErrors:    true,
			expectedField: "target_url",
		},
		{
			name:          "異常系: URL としてパースできない文字列",
			input:         LinkDataFetcherValidatorInput{TargetURL: "http://exa mple.com"},
			wantErrors:    true,
			expectedField: "target_url",
		},
		{
			name:          "異常系: ホストが空の URL",
			input:         LinkDataFetcherValidatorInput{TargetURL: "https://"},
			wantErrors:    true,
			expectedField: "target_url",
		},
	}

	v := NewLinkDataFetcherValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			err := v.Validate(ctx, tt.input)

			if tt.wantErrors {
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Fatal("エラーが期待されたが、エラーがありません")
				}
				if tt.expectedField != "" && !ve.HasFieldError(tt.expectedField) {
					t.Errorf("フィールド %q のエラーが期待されましたが、ありません", tt.expectedField)
				}
			} else {
				if err != nil {
					t.Errorf("エラーが期待されなかったが、エラーがあります: %v", err)
				}
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		valid bool
	}{
		{"http://example.com", true},
		{"https://example.com/path?query=1", true},
		{"https://www.youtube.com/watch?v=abc", true},
		{"", false},
		{"example.com", false},
		{"https://", false},
		{"javascript:alert(1)", false},
		{"http://exa mple.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			if got := IsValidURL(tt.value); got != tt.valid {
				t.Errorf("IsValidURL(%q) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}
