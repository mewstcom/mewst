package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
)

// variationPostMonths is how many months the variation posts are spread over,
// counted back from the month before the run's own.
//
// They are spread rather than gathered because the export writes one HTML file
// per month: variations kept in a single month could be checked by opening
// that one file, and what is worth checking is that every month's file renders
// a body the same way.
//
// Four is where the spreading stops. A variation per month would reach across
// fourteen files, and an output contract nobody opens far enough to see is not
// one that gets checked.
//
// [Ja] variationPostMonths は、エクスポート確認用のポストを何か月へ散らすか。
// 実行自身の月の 1 つ前から数える。
//
// 1 か月へまとめず散らすのは、エクスポートが月ごとに 1 つの HTML ファイルを書く
// ため。1 か月に収めた確認用ポストは、そのファイルを 1 つ開けば済んでしまう。
// 確かめたいのは、どの月のファイルも本文を同じように描画することである。
//
// 散らすのを 4 か月で止めているのは、月ごとに 1 件ずつとすれば 14 個のファイルに
// またがることになるため。そこまで開かれない出力契約は、確認されない出力契約と
// 変わらない。
const variationPostMonths = 4

// postVariation is one post written for what the export's HTML does with a
// body, as against the everyday posts, which are written for what a month's
// listing does with a count.
//
// [Ja] postVariation は、エクスポートの HTML が本文をどう扱うのかを確かめるために
// 書くポスト 1 件。月ごとの一覧が件数をどう扱うのかを確かめるために書く日常ポストに
// 対するもの。
type postVariation struct {
	// body is the text as it is stored. It is what the archive has to give
	// back unchanged, so it is written here in the form it is checked in,
	// including the spaces and the newlines at either end.
	//
	// [Ja] body は保存されるとおりの本文。アーカイブがそのまま返さなければ
	// ならないものであるため、両端の空白や改行も含め、確認する形のままここに書く。
	body string

	// discarded says whether the post is deleted once it is written. A deleted
	// post is one the archive has to leave out, and the only way to see that
	// it is left out is for one to exist.
	//
	// [Ja] discarded は、書き込んだあとにそのポストを削除するかどうか。削除された
	// ポストはアーカイブが除外しなければならないものであり、除外されていることを
	// 見るには、そうしたポストが存在している必要がある。
	discarded bool
}

// postVariations are the bodies the export's output contract is read against.
// They go to roleMain, because that is the account the archive is generated
// from, and a body written to be looked at is only worth writing where
// somebody is already looking.
//
// Each one is a body the application would have accepted: none is blank once
// trimmed, and none is longer than model.MaximumPostContentLength. A seed that
// wrote a body no account could have posted would be showing how the archive
// renders something it will never be given.
//
// [Ja] postVariations は、エクスポートの出力契約を読み合わせる本文の一覧。
// roleMain へ置く。アーカイブが生成されるのはそのアカウントからであり、見られる
// ために書いた本文は、すでに人が見ている場所にあってはじめて書く意味を持つ。
//
// いずれもアプリケーションが受け付けたはずの本文である。トリムして空になるものは
// なく、model.MaximumPostContentLength より長いものもない。どのアカウントも投稿
// しえない本文を書くシードは、アーカイブが決して与えられないものをどう描画するのかを
// 見せることになる。
var postVariations = []postVariation{
	{
		// Tags and an ampersand. templ escapes what it interpolates, so what
		// the archive has to show is the text itself rather than a bold word
		// and a script that ran.
		//
		// [Ja] タグとアンパサンド。templ は埋め込む値をエスケープするため、
		// アーカイブが見せるべきなのは、太字になった単語や実行されたスクリプトでは
		// なく、本文そのものである。
		body: `<script>alert("こんにちは")</script> と & と <b>太字</b> を、そのまま本文として書いたポスト。`,
	},
	{
		// The characters an escape can be got wrong one at a time: the angle
		// brackets, the ampersand that a double escape turns into &amp;amp;,
		// and both kinds of quote.
		//
		// [Ja] エスケープを 1 文字ずつ取り違えうる文字。不等号と、二重エスケープで
		// &amp;amp; になるアンパサンドと、2 種類の引用符。
		body: `5 < 10 && 10 > 5 は真。'引用符' も "二重引用符" も、本文の一部として書く。`,
	},
	{
		// Newlines, including an empty line. The archive keeps them with
		// white-space: pre-wrap rather than converting them to <br>, so a
		// paragraph that lost its blank line would read as one block.
		//
		// [Ja] 空行を含む改行。アーカイブはこれを <br> へ変換せず
		// white-space: pre-wrap で保つ。空行を失った本文は、ひとつながりの
		// かたまりとして読まれることになる。
		body: "買うもの\n・コーヒー豆\n・ペーパーフィルター\n・牛乳\n\n明日の朝までに。",
	},
	{
		// Spaces at either end. The application stores a body as it was
		// written and only trims to decide whether it is blank, so these
		// spaces are part of the text a parser recovers from .e-content.
		//
		// [Ja] 両端の空白。アプリケーションは本文を書かれたまま保存し、トリムするのは
		// 空かどうかを判定するときだけである。この空白は、パーサーが .e-content から
		// 復元するテキストの一部になる。
		body: "  前後に半角空白を置いたポスト。  ",
	},
	{
		// Newlines at either end verify that .e-content retains them in its
		// textContent and displays them with white-space: pre-wrap.
		//
		// [Ja] 両端の改行が .e-content の textContent に残り、
		// white-space: pre-wrap で表示されることを確認する。
		body: "\n改行で始まり、改行で終わるポスト。\n",
	},
	{
		// A tab and an ideographic space, which are whitespace that a reader
		// can see. pre-wrap is what keeps a tab a tab rather than a single
		// space.
		//
		// [Ja] タブと全角空白。読み手に見える空白である。タブを 1 つの半角空白では
		// なくタブのまま残すのは pre-wrap である。
		body: "タブ\tと全角空白　を挟んだポスト。",
	},
	{
		// Emoji next to Japanese and fullwidth punctuation. The family is a
		// sequence of several code points joined with zero-width joiners, so
		// a byte-wise or UTF-16 mishandling breaks it into separate people.
		//
		// [Ja] 日本語と全角の約物に並ぶ絵文字。家族の絵文字はゼロ幅接合子で
		// 繋いだ複数のコードポイントの列であるため、バイト単位や UTF-16 での
		// 取り違えは、これをばらばらの人物に割ってしまう。
		body: "花見日和 🌸 桜の下でコーヒーを飲んだ ☕ 写真も撮った 📷 家族とも会えた 👨‍👩‍👧 全角の（かっこ）や―ダッシュも混ぜる。",
	},
	{
		// A URL with a query string. The archive shows a body as text, so this
		// is not a link, and the ampersand between the parameters is another
		// place an escape has to hold.
		//
		// [Ja] クエリ文字列付きの URL。アーカイブは本文をテキストとして見せるため、
		// これはリンクにならない。パラメータの間のアンパサンドは、エスケープが
		// 成り立っていなければならないもうひとつの場所である。
		body: "参考にした記事はこれ https://example.com/articles/coffee?utm_source=mewst&page=2 。本文の URL は自動でリンクにならない。",
	},
	{
		// The longest body an account can post. The boundary is written
		// exactly rather than approached, because a column or a renderer that
		// cuts a body short cuts it at the boundary and nowhere else.
		//
		// [Ja] アカウントが投稿できる最も長い本文。境界へ近づけるのではなく
		// ちょうどに書くのは、本文を切り詰めるカラムや描画が、切るとすればその境界で
		// 切るからである。
		body: "長い本文が固定幅の中でどのように折り返されるのかを確かめるためのポスト。エクスポートした HTML は本文の幅を 40rem までに制限しているため、句読点の位置や英数字の混ざり方によって行の切れ目が変わる。最大文字数ちょうどまで書いて、枠からはみ出さないことを見る。狭い画面で読んだときの見え方も、一緒に確かめられる。",
	},
	{
		// A long run with nowhere to break. The archive sets
		// overflow-wrap: anywhere for this: without it, a string like this one
		// widens the column past the screen and every other post with it.
		//
		// [Ja] 折り返せる箇所を持たない長い連なり。アーカイブが
		// overflow-wrap: anywhere を指定しているのはこのためで、指定が無ければ、
		// このような文字列は段を画面の外まで広げ、他のすべてのポストを道連れにする。
		body: "折り返せる空白を持たない長い文字列。https://example.com/a/very/long/path/that/never/offers/a/place/to/break/the/line/2026/09/06/post-in-a-fixed-width-column",
	},
	{
		// One character. An article is laid out around a timestamp and a body,
		// and the shortest body there can be is where that layout either holds
		// or collapses.
		//
		// [Ja] 1 文字。article は日時と本文をもとに組み立てられており、ありうる
		// 最も短い本文は、その組み立てが保たれるか崩れるかの分かれ目になる。
		body: "あ",
	},
	{
		// Deleted, with an everyday body. It is the plain case of what the
		// archive leaves out.
		//
		// [Ja] 削除済み、日常の本文。アーカイブが除外するものの、飾らない形。
		body:      "この本文は削除済みのポスト。アーカイブには出てこない。",
		discarded: true,
	},
	{
		// Deleted, with characters that would be unmistakable in the archive.
		// If the snapshot ever stopped filtering, this is the body that says
		// so at a glance rather than one that reads like the rest.
		//
		// [Ja] 削除済み、アーカイブの中で見間違えようのない文字を含む本文。
		// snapshot が絞り込みをやめてしまったとき、他と同じように読める本文ではなく、
		// この本文がそれを一目で告げる。
		body:      "削除済みの本文に <b>タグ</b> と & を混ぜておく。アーカイブに出てしまえばすぐ分かる。",
		discarded: true,
	},
	{
		// Deleted, across several lines. A month whose count is one too many
		// is easiest to spot where the extra entry takes up several lines.
		//
		// [Ja] 削除済み、複数行。件数が 1 件多い月は、余分なエントリが数行を占めて
		// いるときにいちばん見つけやすい。
		body:      "削除済みの複数行のポスト。\n2 行目。\n3 行目。",
		discarded: true,
	},
}

// writeVariationPosts writes the posts the export's output contract is read
// against, spread over the months before the one the run happens in.
//
// The month the run happens in is left out. It is the month a profile is
// opened at, and what should be at the top of that screen is what the account
// writes every day rather than a body that exists to be difficult.
//
// [Ja] writeVariationPosts は、エクスポートの出力契約を読み合わせるポストを、
// 実行が行われている月より前の月々へ散らして書き込む。
//
// 実行が行われている月は外す。プロフィールを開いたときに見えるのはその月であり、
// その画面の先頭にあるべきなのは、扱いにくくあるために存在する本文ではなく、
// そのアカウントが日々書いているものである。
func (w *postWriter) writeVariationPosts(ctx context.Context, account seedAccount) error {
	// The zone is the account's own, for the same reason the everyday posts
	// use it: which month a post falls into is what the archive splits on.
	//
	// [Ja] タイムゾーンはアカウント自身のものを使う。日常ポストと同じ理由で、
	// ポストがどの月に入るのかはアーカイブが分割する基準であるため。
	location, err := time.LoadLocation(account.user.TimeZone)
	if err != nil {
		return fmt.Errorf("タイムゾーン %q の読み込みに失敗: %w", account.user.TimeZone, err)
	}

	for month, variations := range variationsByMonth() {
		// The span is one month longer than the variations are spread over,
		// and the newest of its windows — the run's own month — is the one
		// index never reaches.
		//
		// [Ja] 期間は散らす月数より 1 か月長く取り、その最も新しい区間 (実行自身の
		// 月) は index が決して届かない区間になる。
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

// variationsByMonth shares the variations out between the months they are
// spread over, dealing them one at a time.
//
// They are dealt rather than cut into blocks so that neighbours in the list
// land in different months. The list is written in order of what each body is
// there to show, so cutting it into blocks would put every escaping case in
// one file and every whitespace case in the next.
//
// [Ja] variationsByMonth は、確認用ポストを、散らす先の月々へ 1 件ずつ配る。
//
// まとめて切り分けず 1 件ずつ配るのは、一覧の中で隣り合うものが別々の月へ落ちる
// ようにするため。一覧は各本文が何を示すためにあるのかの順に書かれているため、
// まとめて切り分けると、エスケープの確認がすべて 1 つのファイルに、空白の確認が
// すべて次のファイルに入ることになる。
func variationsByMonth() [][]postVariation {
	months := make([][]postVariation, variationPostMonths)
	for index, variation := range postVariations {
		month := index % variationPostMonths
		months[month] = append(months[month], variation)
	}

	return months
}

// discardPost marks a post as deleted, which is the state the archive has to
// leave out.
//
// The write is here rather than behind a repository method for the same reason
// discarding a profile is: deleting a post is something Rails does, and a
// method only the seed calls would be a way into that state the Go version
// does not otherwise have.
//
// [Ja] discardPost は、ポストを削除済みにする。アーカイブが除外しなければならない
// のがその状態である。
//
// この書き込みをリポジトリのメソッドではなくここに置くのは、プロフィールの削除と
// 同じ理由による。ポストの削除は Rails が行う操作であり、シードだけが呼ぶメソッドは、
// Go 版がほかに持たない経路をその状態へ向けて開けることになる。
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
