package exports_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/exports"
)

// renderIndex renders the table of contents in one locale.
//
// [Ja] renderIndex は目次を 1 つのロケールで描画する。
func renderIndex(t *testing.T, locale string, data exports.IndexData) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), locale)

	var buf bytes.Buffer
	if err := exports.IndexMain(data).Render(ctx, &buf); err != nil {
		t.Fatalf("目次の描画に失敗: %v", err)
	}
	return buf.String()
}

// newIndexMonth builds one month of the table of contents.
//
// [Ja] newIndexMonth は目次に載せる月を 1 つ組み立てる。
func newIndexMonth(year int, month time.Month, postCount int64) exports.IndexMonth {
	return exports.IndexMonth{
		EntryName:  "posts/" + time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006-01") + ".html",
		MonthStart: time.Date(year, month, 1, 0, 0, 0, 0, time.UTC),
		PostCount:  postCount,
	}
}

func TestIndexMain_WritesItsTextInTheUsersLanguage(t *testing.T) {
	t.Parallel()

	data := exports.IndexData{Months: []exports.IndexMonth{
		newIndexMonth(2026, time.June, 1),
		newIndexMonth(2026, time.July, 12),
	}}

	tests := []struct {
		name   string
		locale string
		// A month reads differently in each language, and English counts its
		// posts in singular and plural.
		//
		// [Ja] 月の読み方は言語ごとに異なり、英語では投稿の件数が単数形と
		// 複数形に分かれる。
		want []string
	}{
		{
			name:   "日本語",
			locale: i18n.LangJa,
			want:   []string{"ポストのエクスポート", "2026年6月のポスト", "1 件", "2026年7月のポスト", "12 件"},
		},
		{
			name:   "英語",
			locale: i18n.LangEn,
			want:   []string{"Post export", "Posts from June 2026", "1 post", "Posts from July 2026", "12 posts"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderIndex(t, tt.locale, data)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("目次に %q が含まれていない: %s", want, got)
				}
			}
		})
	}
}

func TestIndexMain_KeepsDeclaredMonthOrder(t *testing.T) {
	t.Parallel()

	got := renderIndex(t, i18n.LangJa, exports.IndexData{Months: []exports.IndexMonth{
		newIndexMonth(2025, time.December, 1),
		newIndexMonth(2026, time.January, 1),
	}})

	// The months are handed over oldest first, and the reader looks for a month
	// in that order, so the list keeps it instead of sorting the labels.
	//
	// [Ja] 月は古い順に渡され、読み手もその順で目的の月を探すため、リストは
	// ラベルを並べ替えず受け取った順を保つ。
	december := strings.Index(got, "2025年12月")
	january := strings.Index(got, "2026年1月")
	if december < 0 || january < 0 {
		t.Fatalf("目次に両方の月が含まれていない: %s", got)
	}
	if december > january {
		t.Errorf("目次の月の順序が渡した順と異なる: %s", got)
	}
}

func TestIndexMain_TellsTheReaderWhenThereAreNoPosts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{name: "日本語", locale: i18n.LangJa, want: "このエクスポートに含まれるポストはありません。"},
		{name: "英語", locale: i18n.LangEn, want: "This export contains no posts."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderIndex(t, tt.locale, exports.IndexData{})
			if !strings.Contains(got, tt.want) {
				t.Errorf("空の目次に %q が含まれていない: %s", tt.want, got)
			}
			if strings.Contains(got, "<ul>") {
				t.Errorf("空の目次にリストが含まれている: %s", got)
			}
		})
	}
}
