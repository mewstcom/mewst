package exportfile

import (
	"context"
	"io"
	"time"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/exports"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// writeIndexHTML writes the table of contents linking every month entry. The
// document is composed of the shell every entry of the archive shares and the
// index's own content, so that the month entries can reuse the same shell
// while being written one fragment at a time.
//
// [Ja] writeIndexHTML は各月のエントリへリンクする目次を書き出す。ドキュメントは
// アーカイブの全エントリで共通の外枠と、index 固有の内容から組み立てる。月の
// エントリがフラグメントごとに書き出されながら、同じ外枠を再利用できるようにする
// ため。
func writeIndexHTML(ctx context.Context, w io.Writer, archive usecase.ExportArchive) error {
	// The archive is written for its owner, so its text follows the user's
	// locale rather than the locale of whoever triggered the generation.
	//
	// [Ja] アーカイブはその持ち主に向けて書くため、文言は生成を起動した者の
	// ロケールではなくユーザーのロケールに従う。
	ctx = i18n.SetLocale(ctx, archive.Locale)

	hw := &htmlWriter{w: w}
	hw.render(ctx, exports.DocumentStart(exports.DocumentData{
		Locale: archive.Locale,
		Title:  exports.IndexTitle(ctx),
	}))
	hw.render(ctx, exports.IndexMain(indexData(archive)))
	hw.render(ctx, exports.DocumentEnd())
	return hw.err
}

// indexData converts the archive into the view data index.html is rendered
// from, so that the template holds no knowledge of the port's types.
//
// [Ja] indexData はアーカイブを index.html の描画に使う view data へ変換する。
// テンプレートが port の型を知らずに済むようにするため。
func indexData(archive usecase.ExportArchive) exports.IndexData {
	months := make([]exports.IndexMonth, 0, len(archive.Months))
	for _, month := range archive.Months {
		months = append(months, exports.IndexMonth{
			EntryName:  monthEntryName(month),
			MonthStart: month.LocalMonthStart,
			PostCount:  month.PostCount,
		})
	}
	return exports.IndexData{Months: months}
}

// writeMonthHeaderHTML opens a month's document. Each month entry is a complete
// document so that it can be opened directly, without the index.
//
// [Ja] writeMonthHeaderHTML は月のドキュメントを開始する。各月のエントリは完結
// した 1 つのドキュメントで、目次を介さず直接開いても成立する。
func writeMonthHeaderHTML(ctx context.Context, w io.Writer, archive usecase.ExportArchive, month usecase.ExportArchiveMonth) error {
	ctx = i18n.SetLocale(ctx, archive.Locale)

	hw := &htmlWriter{w: w}
	hw.render(ctx, exports.MonthStart(exports.MonthData{
		Locale:     archive.Locale,
		Title:      exports.MonthTitle(ctx, month.LocalMonthStart),
		MonthStart: month.LocalMonthStart,
	}))
	return hw.err
}

// writePostHTML writes one post. The timestamp is converted to the archive's
// zone here, so that the template renders the wall clock of the zone the
// months were computed in and the entry a post lands in agrees with the date
// shown next to it.
//
// [Ja] writePostHTML は投稿を 1 件書き出す。日時をアーカイブのゾーンへ変換する
// のはこの関数で、テンプレートは月を算出したゾーンの壁時計を描画する。投稿が
// 入るエントリと、その隣に表示される日付が食い違わないようにするため。
func writePostHTML(ctx context.Context, w io.Writer, archive usecase.ExportArchive, post usecase.ExportArchivePost) error {
	ctx = i18n.SetLocale(ctx, archive.Locale)

	hw := &htmlWriter{w: w}
	hw.render(ctx, exports.MonthPost(exports.MonthPostData{
		ID:          post.ID,
		Content:     post.Content,
		PublishedAt: post.PublishedAt.In(archiveLocation(archive)),
	}))
	return hw.err
}

// writeMonthFooterHTML closes a month's document.
//
// [Ja] writeMonthFooterHTML は月のドキュメントを閉じる。
func writeMonthFooterHTML(w io.Writer) error {
	// The closing markup carries no text of its own, so it renders the same
	// whatever the locale and the caller's context adds nothing. The port's
	// Close carries no context either.
	//
	// [Ja] 閉じるマークアップは自身の文言を持たないため、ロケールに依らず同じ
	// 出力になり、呼び出し側の context を渡す意味がない。port の Close も
	// context を持たない。
	hw := &htmlWriter{w: w}
	hw.render(context.Background(), exports.MonthEnd())
	return hw.err
}

// htmlWriter keeps the first write error so a document can be emitted as a
// sequence of fragments without an error check between them. Once a write
// fails, later fragments are skipped: the entry is already broken, and the
// caller stops on the returned error.
//
// [Ja] htmlWriter は最初の書き込みエラーを保持し、ドキュメントをフラグメントの
// 列として書き出す間のエラーチェックを不要にする。一度失敗した後のフラグメントは
// 書き出さない。そのエントリは既に壊れており、呼び出し側は返るエラーで処理を
// 止めるため。
type htmlWriter struct {
	w   io.Writer
	err error
}

// render writes one templ component, keeping the first error like the other
// fragments do.
//
// [Ja] render は templ のコンポーネントを 1 つ書き出す。他のフラグメントと同じく
// 最初のエラーを保持する。
func (h *htmlWriter) render(ctx context.Context, component templ.Component) {
	if h.err != nil {
		return
	}
	h.err = component.Render(ctx, h.w)
}

// archiveLocation returns the zone timestamps are rendered in. This builder
// treats a nil location as UTC, matching the port contract and the repository
// listings that produce the months.
//
// [Ja] archiveLocation は日時を描画するゾーンを返す。この builder は port の契約と
// 月を返す repository の一覧に揃え、nil の location を UTC として扱う。
func archiveLocation(archive usecase.ExportArchive) *time.Location {
	if archive.Location == nil {
		return time.UTC
	}
	return archive.Location
}
