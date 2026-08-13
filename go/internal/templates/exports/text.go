package exports

import (
	"context"
	"time"

	"github.com/mewstcom/mewst/go/internal/templates"
)

// clockLayout is the time of day of a post's visible timestamp. It is written
// on a 24-hour clock in every locale, so that a time cannot be read as its
// twelve hours earlier or later counterpart.
//
// [Ja] clockLayout はポストの可視表記における時刻の書式。どのロケールでも
// 24 時間表記で書き、時刻が 12 時間前後の別の時刻として読まれないようにする。
const clockLayout = "15:04"

// IndexTitle returns the title of index.html. The archive's text is written
// for its owner, so every string here is resolved in the user's locale, which
// the caller puts on ctx.
//
// [Ja] IndexTitle は index.html のタイトルを返す。アーカイブの文言はその持ち主に
// 向けて書くため、ここで解決する文字列はすべて、呼び出し側が ctx に載せた
// ユーザーのロケールに従う。
func IndexTitle(ctx context.Context) string {
	return templates.T(ctx, "export_archive_index_title")
}

// MonthTitle returns the title of a month's file.
//
// [Ja] MonthTitle は月のファイルのタイトルを返す。
func MonthTitle(ctx context.Context, monthStart time.Time) string {
	return templates.T(ctx, "export_archive_month_title", map[string]any{
		"MonthLabel": monthLabel(ctx, monthStart),
	})
}

// monthLinkText returns the text of the link to a month's file.
//
// [Ja] monthLinkText は月のファイルへのリンクのテキストを返す。
func monthLinkText(ctx context.Context, monthStart time.Time) string {
	return templates.T(ctx, "export_archive_index_month_link", map[string]any{
		"MonthLabel": monthLabel(ctx, monthStart),
	})
}

// monthHeadingText returns the heading of a month's file. It is a key of its
// own rather than the link text of the table of contents, so that the wording
// of the link can change without changing the heading it leads to.
//
// [Ja] monthHeadingText は月のファイルの見出しを返す。目次のリンクテキストとは
// 別のキーにすることで、リンクの文言を変えてもその行き先の見出しが変わらない
// ようにする。
func monthHeadingText(ctx context.Context, monthStart time.Time) string {
	return templates.T(ctx, "export_archive_month_heading", map[string]any{
		"MonthLabel": monthLabel(ctx, monthStart),
	})
}

// formatCommentText returns the sentence that tells a parser how a post is
// written. The format version is interpolated rather than written into each
// translation, so that a new version cannot be recorded in the meta element
// while a translation still names the old one.
//
// [Ja] formatCommentText は、ポストがどう書かれているかをパーサーへ伝える文を
// 返す。format version は各翻訳に直接書かず埋め込む。新しいバージョンが meta
// 要素にだけ記録され、翻訳側が古いバージョンを名乗る状態を作らないため。
func formatCommentText(ctx context.Context) string {
	return templates.T(ctx, "export_archive_format_comment", map[string]any{
		"Version": FormatVersion,
	})
}

// postPublishedAtText returns a post's published time as the reader's locale
// writes it. publishedAt is already in the archive's zone, so its fields are
// read as the wall clock the reader posted at.
//
// [Ja] postPublishedAtText はポストの投稿日時を、読み手のロケールの書き方で
// 返す。publishedAt はアーカイブのゾーンに変換済みのため、その各フィールドは
// 読み手が投稿した壁時計として読む。
func postPublishedAtText(ctx context.Context, publishedAt time.Time) string {
	return templates.T(ctx, "export_archive_post_published_at", map[string]any{
		"Year":      publishedAt.Year(),
		"Month":     int(publishedAt.Month()),
		"MonthName": publishedAt.Month().String(),
		"Day":       publishedAt.Day(),
		"Time":      publishedAt.Format(clockLayout),
	})
}

// postPublishedAtMachine returns a post's published time for the datetime
// attribute. The offset is kept so that the instant stays unambiguous once the
// file is read outside the reader's own time zone.
//
// [Ja] postPublishedAtMachine は datetime 属性に書くポストの投稿日時を返す。
// ファイルが読み手自身のタイムゾーンの外で読まれても時点が一意に定まるよう、
// オフセットを保つ。
func postPublishedAtMachine(publishedAt time.Time) string {
	return publishedAt.Format(time.RFC3339)
}

// postCountText returns how many posts a month holds. The count is passed as
// an int because that is the type the translation layer reads a plural count
// from.
//
// [Ja] postCountText は月が持つ投稿の件数を返す。翻訳層が複数形の判定に読む
// 型に合わせて、件数は int で渡す。
func postCountText(ctx context.Context, postCount int64) string {
	return templates.T(ctx, "export_archive_index_post_count", map[string]any{
		"Count": int(postCount),
	})
}

// monthLabel returns a calendar month as the reader's locale writes it. Each
// language picks the fields it needs: the year and the month number read
// naturally in Japanese, while English reads the month's name.
//
// [Ja] monthLabel は暦月を、読み手のロケールの書き方で返す。各言語は必要な
// フィールドを選ぶ。日本語では年と月の数字が自然に読め、英語では月の名前を読む。
func monthLabel(ctx context.Context, monthStart time.Time) string {
	return templates.T(ctx, "export_archive_month_label", map[string]any{
		"Year":      monthStart.Year(),
		"Month":     int(monthStart.Month()),
		"MonthName": monthStart.Month().String(),
	})
}
