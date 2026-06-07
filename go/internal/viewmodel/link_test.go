package viewmodel

import (
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
)

func TestNewLink(t *testing.T) {
	t.Parallel()

	link := &model.Link{
		CanonicalURL: "https://example.com/articles/1",
		Domain:       "example.com",
		Title:        "Example Article",
		ImageURL:     "https://example.com/og.png",
	}

	got := NewLink(link)

	if got.CanonicalURL != link.CanonicalURL {
		t.Errorf("CanonicalURL = %q, want %q", got.CanonicalURL, link.CanonicalURL)
	}
	if got.Domain != link.Domain {
		t.Errorf("Domain = %q, want %q", got.Domain, link.Domain)
	}
	if got.Title != link.Title {
		t.Errorf("Title = %q, want %q", got.Title, link.Title)
	}
	if got.ImageURL != link.ImageURL {
		t.Errorf("ImageURL = %q, want %q", got.ImageURL, link.ImageURL)
	}
}

func TestShortenHostAndPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rawURL string
		want   string
	}{
		{
			name:   "25 文字以下の host + path はそのまま返す",
			rawURL: "https://example.com/a",
			want:   "example.com/a",
		},
		{
			name: "ちょうど 25 文字は切り詰めない",
			// host + path = "example.com/abcdefghijklm" (25 runes)
			// [Ja] host + path はちょうど 25 rune
			rawURL: "https://example.com/abcdefghijklm",
			want:   "example.com/abcdefghijklm",
		},
		{
			name: "25 文字を超えると省略記号込みで 25 文字に切り詰める",
			// host + path = "example.com/articles/awesome-post" (33 runes)
			// [Ja] 33 rune なので先頭 22 rune + "..." になる
			rawURL: "https://example.com/articles/awesome-post",
			want:   "example.com/articles/a...",
		},
		{
			name:   "ポートは host に含めない",
			rawURL: "https://example.com:8080/a",
			want:   "example.com/a",
		},
		{
			name:   "クエリパラメータは含めない",
			rawURL: "https://example.com/a?utm_source=x",
			want:   "example.com/a",
		},
		{
			name:   "host が無い値は空文字列を返す",
			rawURL: "not-a-url",
			want:   "",
		},
		{
			name:   "パース不能な値は空文字列を返す",
			rawURL: "https://exa mple.com/",
			want:   "",
		},
		{
			name:   "空文字列は空文字列を返す",
			rawURL: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ShortenHostAndPath(tt.rawURL); got != tt.want {
				t.Errorf("ShortenHostAndPath(%q) = %q, want %q", tt.rawURL, got, tt.want)
			}
		})
	}
}
