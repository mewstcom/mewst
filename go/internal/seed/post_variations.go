package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
)

// variationPostMonthsは、エクスポート確認用のポストを何か月へ散らすか。
// 実行自身の月の1つ前から数える。
//
// 1か月へまとめず散らすのは、エクスポートが月ごとに1つのHTMLファイルを書く
// ため。1か月に収めた確認用ポストは、そのファイルを1つ開けば済んでしまう。
// 確かめたいのは、どの月のファイルも本文を同じように描画することである。
//
// 散らすのを4か月で止めているのは、月ごとに1件ずつとすれば14個のファイルに
// またがることになるため。そこまで開かれない出力契約は、確認されない出力契約と
// 変わらない。
const variationPostMonths = 4

// postVariationは、エクスポートのHTMLが本文をどう扱うのかを確かめるために
// 書くポスト1件。月ごとの一覧が件数をどう扱うのかを確かめるために書く日常ポストに
// 対するもの。
type postVariation struct {
	// bodyは保存されるとおりの本文。アーカイブがそのまま返さなければ
	// ならないものであるため、両端の空白や改行も含め、確認する形のままここに書く。
	body string

	// discardedは、書き込んだあとにそのポストを削除するかどうか。削除された
	// ポストはアーカイブが除外しなければならないものであり、除外されていることを
	// 見るには、そうしたポストが存在している必要がある。
	discarded bool
}

// postVariationsは、エクスポートの出力契約を読み合わせる本文の一覧。
// roleMainへ置く。アーカイブが生成されるのはそのアカウントからであり、見られる
// ために書いた本文は、すでに人が見ている場所にあってはじめて書く意味を持つ。
//
// いずれもアプリケーションが受け付けたはずの本文である。トリムして空になるものは
// なく、model.MaximumPostContentLengthより長いものもない。どのアカウントも投稿
// しえない本文を書くシードは、アーカイブが決して与えられないものをどう描画するのかを
// 見せることになる。
var postVariations = []postVariation{
	{
		// タグとアンパサンド。templは埋め込む値をエスケープするため、
		// アーカイブが見せるべきなのは、太字になった単語や実行されたスクリプトでは
		// なく、本文そのものである。
		body: `<script>alert("こんにちは")</script> と & と <b>太字</b> を、そのまま本文として書いたポスト。`,
	},
	{
		// エスケープを1文字ずつ取り違えうる文字。不等号と、二重エスケープで
		// &amp;amp; になるアンパサンドと、2種類の引用符。
		body: `5 < 10 && 10 > 5 は真。'引用符' も "二重引用符" も、本文の一部として書く。`,
	},
	{
		// 空行を含む改行。アーカイブはこれを <br> へ変換せず
		// white-space: pre-wrapで保つ。空行を失った本文は、ひとつながりの
		// かたまりとして読まれることになる。
		body: "買うもの\n・コーヒー豆\n・ペーパーフィルター\n・牛乳\n\n明日の朝までに。",
	},
	{
		// 両端の空白。アプリケーションは本文を書かれたまま保存し、トリムするのは
		// 空かどうかを判定するときだけである。この空白は、パーサーが .e-contentから
		// 復元するテキストの一部になる。
		body: "  前後に半角空白を置いたポスト。  ",
	},
	{
		// 両端の改行が .e-contentのtextContentに残り、
		// white-space: pre-wrapで表示されることを確認する。
		body: "\n改行で始まり、改行で終わるポスト。\n",
	},
	{
		// タブと全角空白。読み手に見える空白である。タブを1つの半角空白では
		// なくタブのまま残すのはpre-wrapである。
		body: "タブ\tと全角空白　を挟んだポスト。",
	},
	{
		// 日本語と全角の約物に並ぶ絵文字。家族の絵文字はゼロ幅接合子で
		// 繋いだ複数のコードポイントの列であるため、バイト単位やUTF-16での
		// 取り違えは、これをばらばらの人物に割ってしまう。
		body: "花見日和 🌸 桜の下でコーヒーを飲んだ ☕ 写真も撮った 📷 家族とも会えた 👨‍👩‍👧 全角の（かっこ）や―ダッシュも混ぜる。",
	},
	{
		// クエリ文字列付きのURL。アーカイブは本文をテキストとして見せるため、
		// これはリンクにならない。パラメータの間のアンパサンドは、エスケープが
		// 成り立っていなければならないもうひとつの場所である。
		body: "参考にした記事はこれ https://example.com/articles/coffee?utm_source=mewst&page=2 。本文の URL は自動でリンクにならない。",
	},
	{
		// アカウントが投稿できる最も長い本文。境界へ近づけるのではなく
		// ちょうどに書くのは、本文を切り詰めるカラムや描画が、切るとすればその境界で
		// 切るからである。
		body: "長い本文が固定幅の中でどのように折り返されるのかを確かめるためのポスト。エクスポートした HTML は本文の幅を 40rem までに制限しているため、句読点の位置や英数字の混ざり方によって行の切れ目が変わる。最大文字数ちょうどまで書いて、枠からはみ出さないことを見る。狭い画面で読んだときの見え方も、一緒に確かめられる。",
	},
	{
		// 折り返せる箇所を持たない長い連なり。アーカイブが
		// overflow-wrap: anywhereを指定しているのはこのためで、指定が無ければ、
		// このような文字列は段を画面の外まで広げ、他のすべてのポストを道連れにする。
		body: "折り返せる空白を持たない長い文字列。https://example.com/a/very/long/path/that/never/offers/a/place/to/break/the/line/2026/09/06/post-in-a-fixed-width-column",
	},
	{
		// 1文字。articleは日時と本文をもとに組み立てられており、ありうる
		// 最も短い本文は、その組み立てが保たれるか崩れるかの分かれ目になる。
		body: "あ",
	},
	{
		// 削除済み、日常の本文。アーカイブが除外するものの、飾らない形。
		body:      "この本文は削除済みのポスト。アーカイブには出てこない。",
		discarded: true,
	},
	{
		// 削除済み、アーカイブの中で見間違えようのない文字を含む本文。
		// snapshotが絞り込みをやめてしまったとき、他と同じように読める本文ではなく、
		// この本文がそれを一目で告げる。
		body:      "削除済みの本文に <b>タグ</b> と & を混ぜておく。アーカイブに出てしまえばすぐ分かる。",
		discarded: true,
	},
	{
		// 削除済み、複数行。件数が1件多い月は、余分なエントリが数行を占めて
		// いるときにいちばん見つけやすい。
		body:      "削除済みの複数行のポスト。\n2 行目。\n3 行目。",
		discarded: true,
	},
}

// writeVariationPostsは、エクスポートの出力契約を読み合わせるポストを、
// 実行が行われている月より前の月々へ散らして書き込む。
//
// 実行が行われている月は外す。プロフィールを開いたときに見えるのはその月であり、
// その画面の先頭にあるべきなのは、扱いにくくあるために存在する本文ではなく、
// そのアカウントが日々書いているものである。
func (w *postWriter) writeVariationPosts(ctx context.Context, account seedAccount) error {
	// タイムゾーンはアカウント自身のものを使う。日常ポストと同じ理由で、
	// ポストがどの月に入るのかはアーカイブが分割する基準であるため。
	location, err := time.LoadLocation(account.user.TimeZone)
	if err != nil {
		return fmt.Errorf("タイムゾーン%qの読み込みに失敗: %w", account.user.TimeZone, err)
	}

	for month, variations := range variationsByMonth() {
		// 期間は散らす月数より1か月長く取り、その最も新しい区間 (実行自身の
		// 月) はindexが決して届かない区間になる。
		start, end := monthWindow(w.now, location, variationPostMonths+1, month)

		for index, variation := range variations {
			publishedAt := storedInstant(postTimeInWindow(start, end, index, len(variations)))

			postID, err := w.writePost(ctx, account, variation.body, publishedAt)
			if err != nil {
				return err
			}

			if variation.discarded {
				if err := w.discardPost(ctx, postID); err != nil {
					return err
				}

				continue
			}

			if err := w.recordLastPostAt(ctx, account, publishedAt); err != nil {
				return err
			}
		}
	}

	return nil
}

// variationsByMonthは、確認用ポストを、散らす先の月々へ1件ずつ配る。
//
// まとめて切り分けず1件ずつ配るのは、一覧の中で隣り合うものが別々の月へ落ちる
// ようにするため。一覧は各本文が何を示すためにあるのかの順に書かれているため、
// まとめて切り分けると、エスケープの確認がすべて1つのファイルに、空白の確認が
// すべて次のファイルに入ることになる。
func variationsByMonth() [][]postVariation {
	months := make([][]postVariation, variationPostMonths)
	for index, variation := range postVariations {
		month := index % variationPostMonths
		months[month] = append(months[month], variation)
	}

	return months
}

// discardPostは、ポストを削除済みにする。アーカイブが除外しなければならない
// のがその状態である。
//
// この書き込みをリポジトリのメソッドではなくここに置くのは、プロフィールの削除と
// 同じ理由による。ポストの削除はRailsが行う操作であり、シードだけが呼ぶメソッドは、
// Go版がほかに持たない経路をその状態へ向けて開けることになる。
func (w *postWriter) discardPost(ctx context.Context, postID model.PostID) error {
	if _, err := w.tx.ExecContext(ctx, `
		UPDATE posts
		SET discarded_at = $2, updated_at = NOW()
		WHERE id = $1
	`, uuid.UUID(postID), storedInstant(w.now)); err != nil {
		return fmt.Errorf("ポストの削除済み化に失敗: %w", err)
	}

	return nil
}
