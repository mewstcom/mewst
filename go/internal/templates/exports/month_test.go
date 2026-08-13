package exports_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/exports"
)

// render renders one fragment of a month's file in one locale.
//
// [Ja] render は月のファイルのフラグメントを 1 つ、1 つのロケールで描画する。
func render(t *testing.T, locale string, component templ.Component) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), locale)

	var buf bytes.Buffer
	if err := component.Render(ctx, &buf); err != nil {
		t.Fatalf("フラグメントの描画に失敗: %v", err)
	}
	return buf.String()
}

// julyStart is the month the fixtures below write. It is a calendar label
// rather than an instant, so it is written as the wall clock of the month's
// first day.
//
// [Ja] julyStart は以下のフィクスチャが書き出す月。時点ではなく暦月のラベルの
// ため、その月の初日の壁時計として書く。
var julyStart = time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

// julyPost is a post published at a time of day that reads differently on a
// 12-hour and a 24-hour clock.
//
// [Ja] julyPost は、12 時間表記と 24 時間表記で読みが分かれる時刻に投稿された
// ポスト。
var julyPost = exports.MonthPostData{
	ID:          "post-1",
	Content:     "7 月のポスト",
	PublishedAt: time.Date(2026, time.July, 23, 21, 30, 0, 0, time.FixedZone("JST", 9*60*60)),
}

func TestMonthStart_WritesItsTextInTheUsersLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		locale string
		// A month reads differently in each language, and so does the sentence
		// that tells a parser how a post is written.
		//
		// [Ja] 月の読み方は言語ごとに異なり、ポストの書かれ方をパーサーへ伝える
		// 文も同じく異なる。
		want []string
	}{
		{
			name:   "日本語",
			locale: i18n.LangJa,
			want:   []string{`<html lang="ja">`, "2026年7月のポスト - Mewst", "<h1>2026年7月のポスト</h1>", "Mewst エクスポート形式 v1"},
		},
		{
			name:   "英語",
			locale: i18n.LangEn,
			want:   []string{`<html lang="en">`, "Posts from July 2026 - Mewst", "<h1>Posts from July 2026</h1>", "Mewst export format v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), tt.locale)
			got := render(t, tt.locale, exports.MonthStart(exports.MonthData{
				Locale:     tt.locale,
				Title:      exports.MonthTitle(ctx, julyStart),
				MonthStart: julyStart,
			}))

			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("月のファイルの冒頭に %q が含まれていない: %s", want, got)
				}
			}
		})
	}
}

func TestMonthPost_WritesThePublishedTimeInTheUsersLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{name: "日本語", locale: i18n.LangJa, want: ">2026年7月23日 21:30</time>"},
		{name: "英語", locale: i18n.LangEn, want: ">July 23, 2026 21:30</time>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := render(t, tt.locale, exports.MonthPost(julyPost))
			if !strings.Contains(got, tt.want) {
				t.Errorf("ポストに %q が含まれていない: %s", tt.want, got)
			}
			// The machine-readable form is the same in every language, so a
			// parser reads one format whatever the reader's locale is.
			//
			// [Ja] 機械可読な表記はどの言語でも同じにする。読み手のロケールに
			// 依らず、パーサーが読む形式が 1 つになるようにするため。
			if want := `datetime="2026-07-23T21:30:00+09:00"`; !strings.Contains(got, want) {
				t.Errorf("ポストに %q が含まれていない: %s", want, got)
			}
		})
	}
}

func TestMonthEnd_ClosesTheDocumentMonthStartOpened(t *testing.T) {
	t.Parallel()

	// The two fragments are written by different calls, so the markup one
	// opens is only balanced if the other closes it.
	//
	// [Ja] 2 つのフラグメントは別々の呼び出しが書き出すため、一方が開いた
	// マークアップはもう一方が閉じてはじめて釣り合う。
	opened := render(t, i18n.LangJa, exports.MonthStart(exports.MonthData{Locale: i18n.LangJa, MonthStart: julyStart}))
	closed := render(t, i18n.LangJa, exports.MonthEnd())

	for _, tag := range []string{"<main>", "<body>", "<html "} {
		if !strings.Contains(opened, tag) {
			t.Errorf("月のファイルの冒頭に %q が含まれていない: %s", tag, opened)
		}
	}
	for _, tag := range []string{"</main>", "</body>", "</html>"} {
		if !strings.Contains(closed, tag) {
			t.Errorf("月のファイルの末尾に %q が含まれていない: %s", tag, closed)
		}
	}
}
