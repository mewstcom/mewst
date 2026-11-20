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

// writeIndexHTMLは各月のエントリへリンクする目次を書き出す。ドキュメントは
// アーカイブの全エントリで共通の外枠と、index固有の内容から組み立てる。月の
// エントリがフラグメントごとに書き出されながら、同じ外枠を再利用できるようにする
// ため。
func writeIndexHTML(ctx context.Context, w io.Writer, archive usecase.ExportArchive) error {
	// アーカイブはその持ち主に向けて書くため、文言は生成を起動した者の
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

// indexDataはアーカイブをindex.htmlの描画に使うview dataへ変換する。
// テンプレートがportの型を知らずに済むようにするため。
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

// writeMonthHeaderHTMLは月のドキュメントを開始する。各月のエントリは完結
// した1つのドキュメントで、目次を介さず直接開いても成立する。
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

// writePostHTMLは投稿を1件書き出す。日時をアーカイブのゾーンへ変換する
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

// writeMonthFooterHTMLは月のドキュメントを閉じる。
func writeMonthFooterHTML(w io.Writer) error {
	// 閉じるマークアップは自身の文言を持たないため、ロケールに依らず同じ
	// 出力になり、呼び出し側のcontextを渡す意味がない。portのCloseも
	// contextを持たない。
	hw := &htmlWriter{w: w}
	hw.render(context.Background(), exports.MonthEnd())
	return hw.err
}

// htmlWriterは最初の書き込みエラーを保持し、ドキュメントをフラグメントの
// 列として書き出す間のエラーチェックを不要にする。一度失敗した後のフラグメントは
// 書き出さない。そのエントリは既に壊れており、呼び出し側は返るエラーで処理を
// 止めるため。
type htmlWriter struct {
	w   io.Writer
	err error
}

// renderはtemplのコンポーネントを1つ書き出す。他のフラグメントと同じく
// 最初のエラーを保持する。
func (h *htmlWriter) render(ctx context.Context, component templ.Component) {
	if h.err != nil {
		return
	}
	h.err = component.Render(ctx, h.w)
}

// archiveLocationは日時を描画するゾーンを返す。このbuilderはportの契約と
// 月を返すrepositoryの一覧に揃え、nilのlocationをUTCとして扱う。
func archiveLocation(archive usecase.ExportArchive) *time.Location {
	if archive.Location == nil {
		return time.UTC
	}
	return archive.Location
}
