package usecase

import (
	"context"
	"io"
	"time"
)

// ExportArchiveBuilderはエクスポートのzipアーカイブを描画する
// Application層port。archive/zipとテンプレートによる実装は
// internal/exportfileに置き、UseCaseはこのinterfaceにだけ依存することで、
// 生成処理が描画側のPresentation層パッケージをimportしないようにする。
type ExportArchiveBuilder interface {
	// NewArchiveはwへ書き出すアーカイブを開始する。返されるwriterは
	// 呼び出しに応じてエントリを順に出力するため、呼び出し側は投稿をDBから
	// 分割取得しながらwをオブジェクトストレージへストリーミングできる。
	NewArchive(w io.Writer, archive ExportArchive) ExportArchiveWriter
}

// ExportArchiveWriterは1つのアーカイブのエントリを、先頭のindex.html、
// 続いて月ごとに1つの順で書き出す。開いている月のエントリは常に1つまでの
// ため、次の月を開く前に現在の月を閉じる。
type ExportArchiveWriter interface {
	// WriteIndexはExportArchiveが宣言する月からindex.htmlを書き出す。
	WriteIndex(ctx context.Context) error

	// OpenMonthはmonthのエントリを開始し、その投稿用のwriterを返す。
	// monthは宣言済みの月のいずれかで、開けるのは1度だけ。これにより
	// index.htmlが存在しないエントリへリンクせず、エントリ名も重複しない。
	OpenMonth(ctx context.Context, month ExportArchiveMonth) (ExportArchiveMonthWriter, error)

	// Closeはアーカイブを完成させる。開いたままの月のエントリは先に
	// 閉じるため、deferしたCloseで途中終了した呼び出し側にも読めるzipが
	// 残る。
	//
	// 完成していないアーカイブはエラーとして返す。これは、Closeの成功を
	// アーカイブの完成と見なす呼び出し側が、壊れたアーカイブを成功として
	// 報告しないようにするためである。indexが無い、宣言した月が書き出されて
	// いない、同じ月が2回宣言されている、月の宣言投稿数と書き出した投稿数が
	// 異なる、エントリの書き出しに失敗した、のいずれかであれば完成していないと
	// 見なす。途中終了した経路ではこのエラーは二次的なもので、中断の原因に
	// なったエラーは呼び出し側だけが持つ。呼び出し側はそちらを一次のエラーと
	// して扱い、Closeのエラーは併記するかログにとどめる。
	//
	// Closeの2回目以降の呼び出しは何もしない。
	Close() error
}

// ExportArchiveMonthWriterは1か月分のエントリへ投稿を追記する。
type ExportArchiveMonthWriter interface {
	// WritePostは開いている月のエントリへ投稿を1件追記する。
	WritePost(ctx context.Context, post ExportArchivePost) error

	// Closeは月のエントリを完成させ、書き出した投稿数が宣言した
	// PostCountと一致することを検証する。2回目以降の呼び出しは何もしない。
	Close() error
}

// ExportArchiveはアーカイブ全体を表す。プリミティブだけを持ち、builderは
// ドメインモデルに依存せずにview dataへ変換できる。
type ExportArchive struct {
	// Localeはアーカイブ自身の文言を書き出すユーザーのロケール。
	Locale string

	// Locationは月を算出した解決済みタイムゾーン。builderは全ての日時を
	// このゾーンで描画するため、表示される日付とエントリが持つ月は常に一致する。
	// このportの契約では、repositoryのfallbackと揃えてnilをUTCとして扱う。
	Location *time.Location

	// Monthsはエクスポートに含まれる月を古い順に列挙する。index.htmlは
	// この一覧から作られ、開けるのはここに含まれる月だけ。
	Months []ExportArchiveMonth

	// GeneratedAtは各zipエントリに記録する。ファイルアプリがzipの
	// エポックではなく実際の日付を表示できるようにするため。
	GeneratedAt time.Time
}

// ExportArchiveMonthはエクスポートに含まれる暦月1つ分。
type ExportArchiveMonth struct {
	// LocalMonthStartはアーカイブのLocationにおける月を指す。時点では
	// なく暦月のラベルなので、変換せずそのまま使う。
	LocalMonthStart time.Time

	// PostCountはその月のエントリが持つ投稿の件数。
	PostCount int64
}

// ExportArchivePostは月のエントリへ描画する投稿1件。
type ExportArchivePost struct {
	// IDは投稿の識別子。外部のパーサーがエントリを元の投稿へ辿れるよう
	// アーカイブへ記録する。
	ID string

	// Contentは投稿本文。エスケープし、改行を保ったまま書き出す。
	Content string

	// PublishedAtは時点を表し、アーカイブのLocationで描画する。
	PublishedAt time.Time
}
