package i18n

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestT_Japanese(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		messageID string
		expected  string
	}{
		{
			name:      "ログインページタイトル",
			messageID: "meta.title.sign_in.new",
			expected:  "ログイン",
		},
		{
			name:      "メールアドレスラベル",
			messageID: "forms.attributes.session_form.email",
			expected:  "メールアドレス",
		},
		{
			name:      "パスワードラベル",
			messageID: "forms.attributes.session_form.password",
			expected:  "パスワード",
		},
		{
			name:      "ログインボタン",
			messageID: "verbs.sign_in",
			expected:  "ログインする",
		},
		{
			name:      "ログイン成功メッセージ",
			messageID: "messages.authentication.sign_in",
			expected:  "ログインしました",
		},
		{
			name:      "必須バリデーションエラー",
			messageID: "errors.validation.required",
			expected:  "入力してください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = SetLocale(ctx, LangJa)

			got := T(ctx, tt.messageID)
			if got != tt.expected {
				t.Errorf("T(ctx, %q) = %q, want %q", tt.messageID, got, tt.expected)
			}
		})
	}
}

func TestT_English(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		messageID string
		expected  string
	}{
		{
			name:      "Sign in page title",
			messageID: "meta.title.sign_in.new",
			expected:  "Sign in",
		},
		{
			name:      "Email label",
			messageID: "forms.attributes.session_form.email",
			expected:  "Email",
		},
		{
			name:      "Password label",
			messageID: "forms.attributes.session_form.password",
			expected:  "Password",
		},
		{
			name:      "Sign in button",
			messageID: "verbs.sign_in",
			expected:  "Sign in",
		},
		{
			name:      "Sign in success message",
			messageID: "messages.authentication.sign_in",
			expected:  "Signed in successfully.",
		},
		{
			name:      "Required validation error",
			messageID: "errors.validation.required",
			expected:  "is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = SetLocale(ctx, LangEn)

			got := T(ctx, tt.messageID)
			if got != tt.expected {
				t.Errorf("T(ctx, %q) = %q, want %q", tt.messageID, got, tt.expected)
			}
		})
	}
}

func TestT_FallbackToMessageID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = SetLocale(ctx, LangJa)

	unknownKey := "unknown.message.key"
	got := T(ctx, unknownKey)
	if got != unknownKey {
		t.Errorf("T(ctx, %q) = %q, want %q", unknownKey, got, unknownKey)
	}
}

func TestT_DefaultLocale(t *testing.T) {
	t.Parallel()

	// ロケールが設定されていない場合、デフォルト（日本語）が使われる
	ctx := context.Background()

	got := T(ctx, "meta.title.sign_in.new")
	expected := "ログイン"
	if got != expected {
		t.Errorf("T(ctx, %q) = %q, want %q (default locale should be Japanese)", "meta.title.sign_in.new", got, expected)
	}
}

func TestGetLocale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locale   string
		expected string
	}{
		{
			name:     "日本語が設定されている場合",
			locale:   LangJa,
			expected: LangJa,
		},
		{
			name:     "英語が設定されている場合",
			locale:   LangEn,
			expected: LangEn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = SetLocale(ctx, tt.locale)

			got := GetLocale(ctx)
			if got != tt.expected {
				t.Errorf("GetLocale(ctx) = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetLocale_Default(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	got := GetLocale(ctx)
	if got != DefaultLang {
		t.Errorf("GetLocale(ctx) = %q, want %q (default)", got, DefaultLang)
	}
}

func TestDetectLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		acceptLanguage string
		expected       string
	}{
		{
			name:           "日本語が含まれている場合",
			acceptLanguage: "ja,en-US;q=0.9,en;q=0.8",
			expected:       LangJa,
		},
		{
			name:           "日本語が最優先の場合",
			acceptLanguage: "ja-JP,ja;q=0.9",
			expected:       LangJa,
		},
		{
			name:           "英語のみの場合",
			acceptLanguage: "en-US,en;q=0.9",
			expected:       LangEn,
		},
		{
			name:           "その他の言語の場合（デフォルト）",
			acceptLanguage: "fr-FR,fr;q=0.9",
			expected:       DefaultLang,
		},
		{
			name:           "空の場合（デフォルト）",
			acceptLanguage: "",
			expected:       DefaultLang,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "/", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}

			got := DetectLanguage(req)
			if got != tt.expected {
				t.Errorf("DetectLanguage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSetLocale_And_GetLocale(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// 日本語を設定
	ctx = SetLocale(ctx, LangJa)
	if got := GetLocale(ctx); got != LangJa {
		t.Errorf("GetLocale() = %q, want %q", got, LangJa)
	}

	// 英語に変更
	ctx = SetLocale(ctx, LangEn)
	if got := GetLocale(ctx); got != LangEn {
		t.Errorf("GetLocale() = %q, want %q", got, LangEn)
	}
}

func TestGetLocalizer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = SetLocale(ctx, LangJa)

	localizer := GetLocalizer(ctx)
	if localizer == nil {
		t.Error("GetLocalizer() returned nil")
	}
}

func TestNewLocalizer(t *testing.T) {
	t.Parallel()

	localizer := NewLocalizer(LangJa)
	if localizer == nil {
		t.Error("NewLocalizer() returned nil")
	}
}
