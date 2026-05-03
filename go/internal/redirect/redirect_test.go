package redirect

import "testing"

func TestValidateBackURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		backURL string
		want    bool
	}{
		{
			name:    "空文字は無効",
			backURL: "",
			want:    false,
		},
		{
			name:    "ルート相対パスは有効",
			backURL: "/home",
			want:    true,
		},
		{
			name:    "ルートパスは有効",
			backURL: "/",
			want:    true,
		},
		{
			name:    "クエリパラメータ付きのルート相対パスは有効",
			backURL: "/search?q=test",
			want:    true,
		},
		{
			name:    "日本語を含むルート相対パスは有効",
			backURL: "/users/テスト",
			want:    true,
		},
		{
			name:    "プロトコル相対URLは無効",
			backURL: "//evil.com",
			want:    false,
		},
		{
			name:    "絶対URLは無効",
			backURL: "https://evil.com",
			want:    false,
		},
		{
			name:    "httpの絶対URLは無効",
			backURL: "http://evil.com",
			want:    false,
		},
		{
			name:    "javascript URLは無効",
			backURL: "javascript:alert(1)",
			want:    false,
		},
		{
			name:    "スラッシュで始まらないパスは無効",
			backURL: "home",
			want:    false,
		},
		{
			name:    "クエリに別ドメインを含むルート相対パスは有効",
			backURL: "/oauth/authorize?client_id=xxx&redirect_uri=https://example.com",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidateBackURL(tt.backURL); got != tt.want {
				t.Errorf("ValidateBackURL(%q) = %v, want %v", tt.backURL, got, tt.want)
			}
		})
	}
}

func TestGetSafeRedirectURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		backURL string
		want    string
	}{
		{
			name:    "有効なURLはそのまま返す",
			backURL: "/home",
			want:    "/home",
		},
		{
			name:    "空文字はデフォルトURL",
			backURL: "",
			want:    "/",
		},
		{
			name:    "危険なURLはデフォルトURL",
			backURL: "//evil.com",
			want:    "/",
		},
		{
			name:    "絶対URLはデフォルトURL",
			backURL: "https://evil.com",
			want:    "/",
		},
		{
			name:    "クエリパラメータ付きはそのまま返す",
			backURL: "/search?q=test",
			want:    "/search?q=test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := GetSafeRedirectURL(tt.backURL); got != tt.want {
				t.Errorf("GetSafeRedirectURL(%q) = %v, want %v", tt.backURL, got, tt.want)
			}
		})
	}
}

func TestAppendSafeBack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		base    string
		backURL string
		want    string
	}{
		{
			name:    "有効なbackURLはエスケープしてクエリに付加",
			base:    "/sign_up",
			backURL: "/settings",
			want:    "/sign_up?back=%2Fsettings",
		},
		{
			name:    "空文字のbackURLはbaseのみ",
			base:    "/sign_up",
			backURL: "",
			want:    "/sign_up",
		},
		{
			name:    "プロトコル相対URLのbackURLはbaseのみ",
			base:    "/sign_up",
			backURL: "//evil.com",
			want:    "/sign_up",
		},
		{
			name:    "絶対URLのbackURLはbaseのみ",
			base:    "/sign_up",
			backURL: "https://evil.com",
			want:    "/sign_up",
		},
		{
			name:    "クエリパラメータを含むbackURLはエスケープして付加",
			base:    "/email_confirmation",
			backURL: "/search?q=test",
			want:    "/email_confirmation?back=%2Fsearch%3Fq%3Dtest",
		},
		{
			name:    "日本語パスのbackURLもエスケープして付加",
			base:    "/sign_up",
			backURL: "/users/テスト",
			want:    "/sign_up?back=%2Fusers%2F%E3%83%86%E3%82%B9%E3%83%88",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := AppendSafeBack(tt.base, tt.backURL); got != tt.want {
				t.Errorf("AppendSafeBack(%q, %q) = %v, want %v", tt.base, tt.backURL, got, tt.want)
			}
		})
	}
}
