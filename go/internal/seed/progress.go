package seed

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// accountsHeadingはアカウントの報告の見出し。
//
// 実行が用意したアカウントを名指しする。報告が読まれるのはそのためである。実行は
// たった今データベースを空にし、その場所へこれらのアカウントを書き込んだのであり、
// この見出しの下の行が、開発者がサインインに使う行になる。
const accountsHeading = "作成したアカウント:"

// progressは、実行が作業しながら自身を説明する手段。
//
// 標準出力ではなくslogが書いているのと同じストリームへ書く。自身の行と、その
// 前後のログ行が、書いた順のまま並ぶようにするため。実行の標準出力を読む利用側は
// 無いため、そこへ書かないことで取り残される呼び出し側はいない。
type progress struct {
	out io.Writer
}

// newProgressはoutへ書き込むprogressを返す。
func newProgress(out io.Writer) *progress {
	return &progress{out: out}
}

// lineは、実行が何をしているのかについての1行を書く。
//
// 書き込みエラーは捨てる。ここは実行が作業のかたわらで行う報告であり、データベース
// を空にして入れ直している最中の実行が、自身を説明していた端末が居なくなったことを
// 理由に止まっても得るものが無いため。
func (p *progress) line(format string, args ...any) {
	_, _ = fmt.Fprintf(p.out, format+"\n", args...)
}

// accountsは、実行が作成したアカウントを、生成器がそれを名指しする役割・
// ページが置かれるatname・サインインに使うアドレス・それぞれが何を確認するために
// いるのかを述べた覚え書きの組で報告する。
//
// 開発者は、どのatnameが何を書いたかを覚えているかどうかではなく、何を見せる
// アカウントなのかでアカウントを選ぶ。覚え書きを名簿ファイルに置いたままにせず、
// アドレスの隣に報告するのはそのためである。
func (p *progress) accounts(accounts []seedAccount) {
	p.line("")
	p.line(accountsHeading)

	// 列は固定数の空白で区切るのではなく揃える。atnameも覚え書きも、その
	// 長さがそのまま長さであり、5件から1件を選ぶために読まれる報告は、列を縦に
	// 追って読まれるため。
	w := tabwriter.NewWriter(p.out, 0, 0, 2, ' ', 0)
	for _, account := range accounts {
		entry := account.roster
		_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", entry.role, entry.atname, entry.email, entry.note)
	}
	_ = w.Flush()
}
