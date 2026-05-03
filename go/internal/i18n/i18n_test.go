package i18n

import (
	"context"
	"net/http"
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
			messageID: "sign_in_title",
			expected:  "Mewstにログイン",
		},
		{
			name:      "メールアドレスラベル",
			messageID: "label_email",
			expected:  "メールアドレス",
		},
		{
			name:      "パスワードラベル",
			messageID: "label_password",
			expected:  "パスワード",
		},
		{
			name:      "ログインボタン",
			messageID: "btn_sign_in",
			expected:  "ログインする",
		},
		{
			name:      "ログイン成功メッセージ",
			messageID: "flash_sign_in_success",
			expected:  "ログインしました",
		},
		{
			name:      "必須バリデーションエラー",
			messageID: "error_required",
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
			messageID: "sign_in_title",
			expected:  "Sign in to Mewst",
		},
		{
			name:      "Email label",
			messageID: "label_email",
			expected:  "Email",
		},
		{
			name:      "Password label",
			messageID: "label_password",
			expected:  "Password",
		},
		{
			name:      "Sign in button",
			messageID: "btn_sign_in",
			expected:  "Sign in",
		},
		{
			name:      "Sign in success message",
			messageID: "flash_sign_in_success",
			expected:  "Signed in successfully.",
		},
		{
			name:      "Required validation error",
			messageID: "error_required",
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

	got := T(ctx, "sign_in_title")
	expected := "Mewstにログイン"
	if got != expected {
		t.Errorf("T(ctx, %q) = %q, want %q (default locale should be Japanese)", "sign_in_title", got, expected)
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

func TestMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		acceptLanguage string
		wantLocale     string
		wantTranslated string
	}{
		{
			name:           "日本語ヘッダー",
			acceptLanguage: "ja",
			wantLocale:     LangJa,
			wantTranslated: "Mewstにログイン",
		},
		{
			name:           "英語ヘッダー",
			acceptLanguage: "en",
			wantLocale:     LangEn,
			wantTranslated: "Sign in to Mewst",
		},
		{
			name:           "ヘッダーなしはデフォルト言語",
			acceptLanguage: "",
			wantLocale:     DefaultLang,
			wantTranslated: "Mewstにログイン",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedLocale string
			var capturedTranslation string

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedLocale = GetLocale(r.Context())
				capturedTranslation = T(r.Context(), "sign_in_title")
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			rr := httptest.NewRecorder()

			Middleware(testHandler).ServeHTTP(rr, req)

			if capturedLocale != tt.wantLocale {
				t.Errorf("Middleware() locale = %q, want %q", capturedLocale, tt.wantLocale)
			}

			if capturedTranslation != tt.wantTranslated {
				t.Errorf("Middleware() translation = %q, want %q", capturedTranslation, tt.wantTranslated)
			}
		})
	}
}
