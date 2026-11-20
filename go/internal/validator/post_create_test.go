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
			// マルチバイト文字はrune単位で数えるため、日本語160文字は有効。
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
			// 空白のみの本文はRailsではblank (presence: true) として弾かれるため、
			// ここでも必須エラーにする。
			name:          "異常系: 半角スペースのみ",
			input:         PostCreateValidatorInput{Content: "   "},
			wantErrors:    true,
			expectedField: "content",
		},
		{
			// 全角スペース (U+3000) もRailsではblank。strings.TrimSpaceは
			// unicode.IsSpace経由でこれを除去する。
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
			// 日本語161文字はruneベースの上限を超える (バイトベースでは更に大きく超える)。
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
					t.Errorf("フィールド%qのエラーが期待されましたが、ありません", tt.expectedField)
				}
			} else {
				if err != nil {
					t.Errorf("エラーが期待されなかったが、エラーがあります: %v", err)
				}
			}
		})
	}
}
