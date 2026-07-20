package validator

import (
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
)

func TestPostCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         PostCreateValidatorInput
		wantErrors    bool
		expectedField string
	}{
		{
			name:       "正常系: 通常の本文",
			input:      PostCreateValidatorInput{Content: "Hello, Mewst!"},
			wantErrors: false,
		},
		{
			name:       "正常系: 1文字",
			input:      PostCreateValidatorInput{Content: "a"},
			wantErrors: false,
		},
		{
			name:       "正常系: 160文字ちょうど",
			input:      PostCreateValidatorInput{Content: strings.Repeat("a", 160)},
			wantErrors: false,
		},
		{
			// Multibyte characters are counted by rune, so 160 Japanese
			// characters are valid.
			// [Ja] マルチバイト文字は rune 単位で数えるため、日本語 160 文字は有効。
			name:       "正常系: 日本語160文字ちょうど",
			input:      PostCreateValidatorInput{Content: strings.Repeat("あ", 160)},
			wantErrors: false,
		},
		{
			name:          "異常系: 本文が空",
			input:         PostCreateValidatorInput{Content: ""},
			wantErrors:    true,
			expectedField: "content",
		},
		{
			// Whitespace-only bodies are blank in Rails (presence: true), so
			// they must be rejected here too.
			// [Ja] 空白のみの本文は Rails では blank (presence: true) として弾かれるため、
			// ここでも必須エラーにする。
			name:          "異常系: 半角スペースのみ",
			input:         PostCreateValidatorInput{Content: "   "},
			wantErrors:    true,
			expectedField: "content",
		},
		{
			// A full-width space (U+3000) is also blank in Rails; strings.TrimSpace
			// removes it via unicode.IsSpace.
			// [Ja] 全角スペース (U+3000) も Rails では blank。strings.TrimSpace は
			// unicode.IsSpace 経由でこれを除去する。
			name:          "異常系: 全角スペースのみ",
			input:         PostCreateValidatorInput{Content: "　　"},
			wantErrors:    true,
			expectedField: "content",
		},
		{
			name:          "異常系: 161文字",
			input:         PostCreateValidatorInput{Content: strings.Repeat("a", 161)},
			wantErrors:    true,
			expectedField: "content",
		},
		{
			// 161 Japanese characters exceed the rune-based limit even though
			// they would be far over any byte-based one.
			// [Ja] 日本語 161 文字は rune ベースの上限を超える (バイトベースでは更に大きく超える)。
			name:          "異常系: 日本語161文字",
			input:         PostCreateValidatorInput{Content: strings.Repeat("あ", 161)},
			wantErrors:    true,
			expectedField: "content",
		},
	}

	v := NewPostCreateValidator()

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
