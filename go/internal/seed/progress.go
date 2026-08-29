package seed

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// accountsHeading introduces the account report.
//
// It names the accounts a run set up, which is what the report is read for:
// the run has just emptied the database and written these accounts in its
// place, and a line under this heading is what a developer signs in with.
//
// [Ja] accountsHeading はアカウントの報告の見出し。
//
// 実行が用意したアカウントを名指しする。報告が読まれるのはそのためである。実行は
// たった今データベースを空にし、その場所へこれらのアカウントを書き込んだのであり、
// この見出しの下の行が、開発者がサインインに使う行になる。
const accountsHeading = "作成したアカウント:"

// progress is how a run describes itself while it works.
//
// It writes to the stream slog writes to rather than to standard output, so
// that its lines and the log lines around them stay in the order they were
// written. Nothing reads the standard output of a run, so there is no caller
// left behind by not writing there.
//
// [Ja] progress は、実行が作業しながら自身を説明する手段。
//
// 標準出力ではなく slog が書いているのと同じストリームへ書く。自身の行と、その
// 前後のログ行が、書いた順のまま並ぶようにするため。実行の標準出力を読む利用側は
// 無いため、そこへ書かないことで取り残される呼び出し側はいない。
type progress struct {
	out io.Writer
}

// newProgress returns a progress that writes to out.
//
// [Ja] newProgress は out へ書き込む progress を返す。
func newProgress(out io.Writer) *progress {
	return &progress{out: out}
}

// line writes one line about what the run is doing.
//
// The write error is discarded: this is the report a run makes on the side of
// its work, and a run that is emptying and refilling a database has nothing to
// gain from stopping because the terminal it was describing itself to went
// away.
//
// [Ja] line は、実行が何をしているのかについての 1 行を書く。
//
// 書き込みエラーは捨てる。ここは実行が作業のかたわらで行う報告であり、データベース
// を空にして入れ直している最中の実行が、自身を説明していた端末が居なくなったことを
// 理由に止まっても得るものが無いため。
func (p *progress) line(format string, args ...any) {
	_, _ = fmt.Fprintf(p.out, format+"\n", args...)
}

// accounts reports the accounts a run created, as the role a generator
// names them by, the atname their pages sit under, the address they sign in
// with and the note that says what each is there to show.
//
// A developer picks an account by what it is meant to show rather than by
// remembering which atname wrote what, which is why the note is reported
// beside the address instead of being left in the roster file.
//
// [Ja] accounts は、実行が作成したアカウントを、生成器がそれを名指しする役割・
// ページが置かれる atname・サインインに使うアドレス・それぞれが何を確認するために
// いるのかを述べた覚え書きの組で報告する。
//
// 開発者は、どの atname が何を書いたかを覚えているかどうかではなく、何を見せる
// アカウントなのかでアカウントを選ぶ。覚え書きを名簿ファイルに置いたままにせず、
// アドレスの隣に報告するのはそのためである。
func (p *progress) accounts(accounts []seedAccount) {
	p.line("")
	p.line(accountsHeading)

	// The columns are aligned rather than separated by a fixed number of
	// spaces: an atname and a note are both as long as they are, and a report
	// that is read to pick one account out of five is read down a column.
	//
	// [Ja] 列は固定数の空白で区切るのではなく揃える。atname も覚え書きも、その
	// 長さがそのまま長さであり、5 件から 1 件を選ぶために読まれる報告は、列を縦に
	// 追って読まれるため。
	w := tabwriter.NewWriter(p.out, 0, 0, 2, ' ', 0)
	for _, account := range accounts {
		entry := account.roster
		_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", entry.role, entry.atname, entry.email, entry.note)
	}
	_ = w.Flush()
}
