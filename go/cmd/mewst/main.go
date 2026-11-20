// mewstコマンドはMewstのコマンドラインエントリーポイント。serveサブコマンドが
// HTTPサーバーを起動し、seedサブコマンドが開発用データベースをシードデータで作り直し、
// devcredsサブコマンドがseedの作成したアカウント1件がサインインに使う資格情報を書き出す。
//
// バイナリ名を、当初に担っていた1つの役割ではなくサービス名にしているのは、この後に
// 増える運用のone-offタスクを、バイナリを1つずつ増やすのではなくサブコマンドとして
// 足せるようにするため。
package main

import (
	"fmt"
	"io"
	"os"
)

// exitUsageは、サブコマンドを指定していない / 未知のサブコマンドを指定した
// コマンドラインに対する終了コード。使用方法の誤りを2、依頼された処理自体の失敗を1
// とするGoツールチェインやgetoptの慣習に従う。
const exitUsage = 2

// commandsはrunが振り分ける先の実装。1つずつ仮引数で受け取るのではなく1つの
// 値にまとめるのは、後から追加するサブコマンドが、すべての呼び出し箇所で増える引数では
// なく、1つのフィールドで済むようにするため。
type commands struct {
	serve    func()
	seed     func()
	devcreds func(stdout io.Writer, role string)
}

func main() {
	os.Exit(run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		commands{serve: runServe, seed: runSeed, devcreds: runDevCredentials},
	))
}

// runはargsをサブコマンドへ振り分け、プロセスの終了コードを返す。プロセス自身の値を
// 直接使わず、引数・2つの出力先・サブコマンドの実装を仮引数で受け取るのは、テスト
// プロセスを終了させたり、サーバーを起動したり、データベースを空にしたりせずに振り分けを
// 検証できるようにするため。
//
// 標準出力を通しているのはdevcredsのためで、その標準出力はscripts/browse.shが読む
// 機械可読な契約になっている。振り分けの中でos.Stdoutを直接掴むと、呼び出し側が解釈する
// 唯一のストリームがテストから触れなくなる。
//
// サブコマンド無しのときにserveへ既定することはしない。何も指定しない実行はusageを
// 表示して失敗するため、各呼び出し箇所がどのサブコマンドを使うのかを明示することになる。
func run(args []string, stdout, stderr io.Writer, cmds commands) int {
	if len(args) == 0 {
		usage(stderr)

		return exitUsage
	}

	switch args[0] {
	case "serve":
		return runWithoutArguments("serve", args[1:], stderr, cmds.serve)
	case "seed":
		return runWithoutArguments("seed", args[1:], stderr, cmds.seed)
	case "devcreds":
		return runWithOneArgument("devcreds", "<role>", args[1:], stderr, func(role string) {
			cmds.devcreds(stdout, role)
		})
	default:
		// 書き込みエラーは意図的に捨てる。ここは診断情報の出力先そのものであり、
		// その書き込みに失敗したことを報告する先が残っていないため。何が起きたかは
		// 終了コードで呼び出し側に伝わる。
		_, _ = fmt.Fprintf(stderr, "unknown subcommand: %q\n\n", args[0])
		usage(stderr)

		return exitUsage
	}
}

// runWithoutArgumentsは、引数を取らないサブコマンドを実行し、その後ろに何かを
// 続けたコマンドラインには、代わりにusageで応答する。
//
// 余分な引数を出力に含める理由は未知のサブコマンドと同じ。拒否されたことだけを知らされた
// コマンドラインからは、そのどの部分が解釈されなかったのかが分からない。ここに並ぶ
// サブコマンドはいずれも引数を取らず、走り出してしまえばどう起動されても見え方が同じで
// あるため、引数を無視すると打ち間違えたフラグには症状が1つも残らない。サーバーは既定の
// 設定で立ち上がり、シードはデータベースを空にする。いずれも何も告げずに。
func runWithoutArguments(name string, rest []string, stderr io.Writer, command func()) int {
	if len(rest) > 0 {
		_, _ = fmt.Fprintf(stderr, "%s takes no arguments: %q\n\n", name, rest)
		usage(stderr)

		return exitUsage
	}

	command()

	return 0
}

// runWithOneArgumentは、引数をちょうど1つ取るサブコマンドを実行し、それを
// 与えなかった / 複数与えたコマンドラインには、代わりにusageで応答する。
//
// 何のための引数なのかを、書かれていたものと並べて出力に含める。このサブコマンドは
// 端末からと同じくらいシェルスクリプトからも呼ばれるためで、変数を取りこぼした
// スクリプトは空文字列を渡し、引用符を取りこぼしたスクリプトは複数の語を渡す。その
// どちらも、コマンドラインを見ただけでは「役割が足りない」とは読み取れない。
func runWithOneArgument(name, argument string, rest []string, stderr io.Writer, command func(string)) int {
	if len(rest) != 1 {
		_, _ = fmt.Fprintf(stderr, "%s takes exactly one argument %s: %q\n\n", name, argument, rest)
		usage(stderr)

		return exitUsage
	}

	command(rest[0])

	return 0
}

// usageは利用可能なサブコマンドの一覧をwに書く。書き込みエラーを捨てる理由は
// runと同じ。
func usage(w io.Writer) {
	_, _ = fmt.Fprint(w, `usage: mewst <command>

commands:
  serve              start the HTTP server
  seed               rebuild the development database from the seed data
  devcreds <role>    print the email address and password of a seeded account
`)
}
