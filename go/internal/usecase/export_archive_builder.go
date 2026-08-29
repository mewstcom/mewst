package usecase

import (
	"context"
	"io"
	"time"
)

// ExportArchiveBuilder is the Application-layer port for rendering an export's
// zip archive. The archive/zip and template implementation lives in
// internal/exportfile; use cases depend only on this interface so that the
// generation flow never imports the Presentation-layer packages it renders
// with.
//
// [Ja] ExportArchiveBuilder はエクスポートの zip アーカイブを描画する
// Application 層 port。archive/zip とテンプレートによる実装は
// internal/exportfile に置き、UseCase はこの interface にだけ依存することで、
// 生成処理が描画側の Presentation 層パッケージを import しないようにする。
type ExportArchiveBuilder interface {
	// NewArchive starts an archive written to w. The returned writer emits each
	// entry as it is driven, so the caller can stream w into the object storage
	// while the posts are still being paged out of the database.
	//
	// [Ja] NewArchive は w へ書き出すアーカイブを開始する。返される writer は
	// 呼び出しに応じてエントリを順に出力するため、呼び出し側は投稿を DB から
	// 分割取得しながら w をオブジェクトストレージへストリーミングできる。
	NewArchive(w io.Writer, archive ExportArchive) ExportArchiveWriter
}

// ExportArchiveWriter writes the entries of one archive: index.html first, then
// one entry per month. At most one month entry is open at a time, so a month
// must be closed before the next one is opened.
//
// [Ja] ExportArchiveWriter は 1 つのアーカイブのエントリを、先頭の index.html、
// 続いて月ごとに 1 つの順で書き出す。開いている月のエントリは常に 1 つまでの
// ため、次の月を開く前に現在の月を閉じる。
type ExportArchiveWriter interface {
	// WriteIndex writes index.html from the months declared in ExportArchive.
	//
	// [Ja] WriteIndex は ExportArchive が宣言する月から index.html を書き出す。
	WriteIndex(ctx context.Context) error

	// OpenMonth starts the entry for month and returns the writer for its posts.
	// month must be one of the declared months and can be opened only once, so
	// index.html never links to a missing entry and no entry name is duplicated.
	//
	// [Ja] OpenMonth は month のエントリを開始し、その投稿用の writer を返す。
	// month は宣言済みの月のいずれかで、開けるのは 1 度だけ。これにより
	// index.html が存在しないエントリへリンクせず、エントリ名も重複しない。
	OpenMonth(ctx context.Context, month ExportArchiveMonth) (ExportArchiveMonthWriter, error)

	// Close finishes the archive. An open month entry is closed first, so a
	// caller that stops early with a deferred Close still leaves a readable
	// zip behind.
	//
	// An archive that is not complete is reported as an error, so that a
	// caller which treats a successful Close as the archive being finished
	// never reports a broken one as a success. An archive is incomplete when
	// it has no index, when a declared month was never written, when a month
	// was declared twice, when a month's declared post count differs from its
	// written post count, or when writing an entry failed. On the path where
	// the caller stopped early this error is a secondary one: the error that
	// stopped it is held only by the caller, which keeps that one as the
	// primary error and joins or logs the error from Close.
	//
	// Calling Close twice is a no-op.
	//
	// [Ja] Close はアーカイブを完成させる。開いたままの月のエントリは先に
	// 閉じるため、defer した Close で途中終了した呼び出し側にも読める zip が
	// 残る。
	//
	// 完成していないアーカイブはエラーとして返す。これは、Close の成功を
	// アーカイブの完成と見なす呼び出し側が、壊れたアーカイブを成功として
	// 報告しないようにするためである。index が無い、宣言した月が書き出されて
	// いない、同じ月が 2 回宣言されている、月の宣言投稿数と書き出した投稿数が
	// 異なる、エントリの書き出しに失敗した、のいずれかであれば完成していないと
	// 見なす。途中終了した経路ではこのエラーは二次的なもので、中断の原因に
	// なったエラーは呼び出し側だけが持つ。呼び出し側はそちらを一次のエラーと
	// して扱い、Close のエラーは併記するかログにとどめる。
	//
	// Close の 2 回目以降の呼び出しは何もしない。
	Close() error
}

// ExportArchiveMonthWriter appends posts to one month's entry.
//
// [Ja] ExportArchiveMonthWriter は 1 か月分のエントリへ投稿を追記する。
type ExportArchiveMonthWriter interface {
	// WritePost appends one post to the open month entry.
	//
	// [Ja] WritePost は開いている月のエントリへ投稿を 1 件追記する。
	WritePost(ctx context.Context, post ExportArchivePost) error

	// Close finishes the month entry and verifies that the number of posts
	// written matches its declared PostCount. Calling it twice is a no-op.
	//
	// [Ja] Close は月のエントリを完成させ、書き出した投稿数が宣言した
	// PostCount と一致することを検証する。2 回目以降の呼び出しは何もしない。
	Close() error
}

// ExportArchive describes an archive as a whole. It carries only primitives so
// that the builder converts it to view data without depending on the domain
// model.
//
// [Ja] ExportArchive はアーカイブ全体を表す。プリミティブだけを持ち、builder は
// ドメインモデルに依存せずに view data へ変換できる。
type ExportArchive struct {
	// Locale is the user's locale the archive's own text is written in.
	//
	// [Ja] Locale はアーカイブ自身の文言を書き出すユーザーのロケール。
	Locale string

	// Location is the resolved time zone the months were computed in. The
	// builder renders every timestamp in it, so the displayed dates and the
	// month an entry holds always agree. This port's contract treats a nil
	// location as UTC, matching the repository's fallback.
	//
	// [Ja] Location は月を算出した解決済みタイムゾーン。builder は全ての日時を
	// このゾーンで描画するため、表示される日付とエントリが持つ月は常に一致する。
	// この port の契約では、repository の fallback と揃えて nil を UTC として扱う。
	Location *time.Location

	// Months lists every month of the export, oldest first. index.html is built
	// from this list, and only these months can be opened.
	//
	// [Ja] Months はエクスポートに含まれる月を古い順に列挙する。index.html は
	// この一覧から作られ、開けるのはここに含まれる月だけ。
	Months []ExportArchiveMonth

	// GeneratedAt is stamped on every zip entry so that file managers show a
	// real date instead of the zip epoch.
	//
	// [Ja] GeneratedAt は各 zip エントリに記録する。ファイルアプリが zip の
	// エポックではなく実際の日付を表示できるようにするため。
	GeneratedAt time.Time
}

// ExportArchiveMonth is one calendar month of an export.
//
// [Ja] ExportArchiveMonth はエクスポートに含まれる暦月 1 つ分。
type ExportArchiveMonth struct {
	// LocalMonthStart names the month in the archive's Location. It is a
	// calendar label rather than an instant, so it is used as written.
	//
	// [Ja] LocalMonthStart はアーカイブの Location における月を指す。時点では
	// なく暦月のラベルなので、変換せずそのまま使う。
	LocalMonthStart time.Time

	// PostCount is the number of posts the month's entry holds.
	//
	// [Ja] PostCount はその月のエントリが持つ投稿の件数。
	PostCount int64
}

// ExportArchivePost is one post rendered into a month's entry.
//
// [Ja] ExportArchivePost は月のエントリへ描画する投稿 1 件。
type ExportArchivePost struct {
	// ID is the post's identifier, recorded in the archive so that an external
	// parser can tie an entry back to the post it came from.
	//
	// [Ja] ID は投稿の識別子。外部のパーサーがエントリを元の投稿へ辿れるよう
	// アーカイブへ記録する。
	ID string

	// Content is the post body, written escaped and with its newlines kept.
	//
	// [Ja] Content は投稿本文。エスケープし、改行を保ったまま書き出す。
	Content string

	// PublishedAt is an instant, rendered in the archive's Location.
	//
	// [Ja] PublishedAt は時点を表し、アーカイブの Location で描画する。
	PublishedAt time.Time
}
