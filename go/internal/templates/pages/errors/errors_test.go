package errors_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/pages/errors"
)

func TestBadGatewayTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locale   string
		expected []string
	}{
		{
			name:   "日本語でレンダリングされる",
			locale: "ja",
			expected: []string{
				"申し訳ございません。現在サービスに接続できません。",
				"しばらくしてから再度お試しください。",
				"ホームに戻る",
				"⚠️",
				`lang="ja"`,
				`href="/"`,
			},
		},
		{
			name:   "英語でレンダリングされる",
			locale: "en",
			expected: []string{
				"Sorry, we are unable to connect to the service.",
				"Please try again later.",
				"Back to home",
				"⚠️",
				`lang="en"`,
				`href="/"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), tt.locale)

			var buf bytes.Buffer
			err := errors.BadGateway().Render(ctx, &buf)
			if err != nil {
				t.Fatalf("テンプレートのレンダリングに失敗: %v", err)
			}

			body := buf.String()
			for _, expected := range tt.expected {
				if !strings.Contains(body, expected) {
					t.Errorf("レスポンスに %q が含まれていません", expected)
				}
			}
		})
	}
}

func TestNotFoundTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locale   string
		expected []string
	}{
		{
			name:   "日本語でレンダリングされる",
			locale: "ja",
			expected: []string{
				"お探しのページは見つかりませんでした",
				"ホームに戻る",
				"🫥",
				`lang="ja"`,
				`href="/"`,
			},
		},
		{
			name:   "英語でレンダリングされる",
			locale: "en",
			expected: []string{
				"The page you are looking for could not be found",
				"Back to home",
				"🫥",
				`lang="en"`,
				`href="/"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), tt.locale)

			var buf bytes.Buffer
			err := errors.NotFound().Render(ctx, &buf)
			if err != nil {
				t.Fatalf("テンプレートのレンダリングに失敗: %v", err)
			}

			body := buf.String()
			for _, expected := range tt.expected {
				if !strings.Contains(body, expected) {
					t.Errorf("レスポンスに %q が含まれていません", expected)
				}
			}
		})
	}
}
