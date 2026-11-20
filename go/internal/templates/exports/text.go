package exports

import (
	"context"
	"time"

	"github.com/mewstcom/mewst/go/internal/templates"
)

// clockLayoutはポストの可視表記における時刻の書式。どのロケールでも
// 24時間表記で書き、時刻が12時間前後の別の時刻として読まれないようにする。
const clockLayout = "15:04"

// IndexTitleはindex.htmlのタイトルを返す。アーカイブの文言はその持ち主に
// 向けて書くため、ここで解決する文字列はすべて、呼び出し側がctxに載せた
// ユーザーのロケールに従う。
func IndexTitle(ctx context.Context) string {
	return templates.T(ctx, "export_archive_index_title")
}

// MonthTitleは月のファイルのタイトルを返す。
func MonthTitle(ctx context.Context, monthStart time.Time) string {
	return templates.T(ctx, "export_archive_month_title", map[string]any{
		"MonthLabel": monthLabel(ctx, monthStart),
	})
}

// monthLinkTextは月のファイルへのリンクのテキストを返す。
func monthLinkText(ctx context.Context, monthStart time.Time) string {
	return templates.T(ctx, "export_archive_index_month_link", map[string]any{
		"MonthLabel": monthLabel(ctx, monthStart),
	})
}

// monthHeadingTextは月のファイルの見出しを返す。目次のリンクテキストとは
// 別のキーにすることで、リンクの文言を変えてもその行き先の見出しが変わらない
// ようにする。
func monthHeadingText(ctx context.Context, monthStart time.Time) string {
	return templates.T(ctx, "export_archive_month_heading", map[string]any{
		"MonthLabel": monthLabel(ctx, monthStart),
	})
}

// formatCommentTextは、ポストがどう書かれているかをパーサーへ伝える文を
// 返す。format versionは各翻訳に直接書かず埋め込む。新しいバージョンがmeta
// 要素にだけ記録され、翻訳側が古いバージョンを名乗る状態を作らないため。
func formatCommentText(ctx context.Context) string {
	return templates.T(ctx, "export_archive_format_comment", map[string]any{
		"Version": FormatVersion,
	})
}

// postPublishedAtTextはポストの投稿日時を、読み手のロケールの書き方で
// 返す。publishedAtはアーカイブのゾーンに変換済みのため、その各フィールドは
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

// postPublishedAtMachineはdatetime属性に書くポストの投稿日時を返す。
// ファイルが読み手自身のタイムゾーンの外で読まれても時点が一意に定まるよう、
// オフセットを保つ。
func postPublishedAtMachine(publishedAt time.Time) string {
	return publishedAt.Format(time.RFC3339)
}

// postCountTextは月が持つ投稿の件数を返す。翻訳層が複数形の判定に読む
// 型に合わせて、件数はintで渡す。
func postCountText(ctx context.Context, postCount int64) string {
	return templates.T(ctx, "export_archive_index_post_count", map[string]any{
		"Count": int(postCount),
	})
}

// monthLabelは暦月を、読み手のロケールの書き方で返す。各言語は必要な
// フィールドを選ぶ。日本語では年と月の数字が自然に読め、英語では月の名前を読む。
func monthLabel(ctx context.Context, monthStart time.Time) string {
	return templates.T(ctx, "export_archive_month_label", map[string]any{
		"Year":      monthStart.Year(),
		"Month":     int(monthStart.Month()),
		"MonthName": monthStart.Month().String(),
	})
}
