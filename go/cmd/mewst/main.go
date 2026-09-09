// Command mewst is Mewst's command-line entry point. The serve subcommand
// starts the HTTP server, the seed subcommand rebuilds a development database
// from the seed data, and the devcreds subcommand writes the credentials one of
// the accounts that seed created signs in with.
//
// The binary is named after the service rather than after the one role it
// started out filling, so that the operational one-off tasks that follow are
// added as subcommands instead of as a binary each.
//
// [Ja] mewst コマンドは Mewst のコマンドラインエントリーポイント。serve サブコマンドが
// HTTP サーバーを起動し、seed サブコマンドが開発用データベースをシードデータで作り直し、
// devcreds サブコマンドが seed の作成したアカウント 1 件がサインインに使う資格情報を書き出す。
//
// バイナリ名を、当初に担っていた 1 つの役割ではなくサービス名にしているのは、この後に
// 増える運用の one-off タスクを、バイナリを 1 つずつ増やすのではなくサブコマンドとして
// 足せるようにするため。
package main

import (
	"fmt"
	"io"
	"os"
)

// exitUsage is the exit code for a command line that names no subcommand or an
// unknown one. It follows the convention of the Go toolchain and of getopt,
// where 2 signals a usage error and 1 signals a failure of the requested work.
//
// [Ja] exitUsage は、サブコマンドを指定していない / 未知のサブコマンドを指定した
// コマンドラインに対する終了コード。使用方法の誤りを 2、依頼された処理自体の失敗を 1
// とする Go ツールチェインや getopt の慣習に従う。
const exitUsage = 2

// commands are the implementations run dispatches to. They are gathered into
// one value rather than taken as a parameter each, so that a subcommand added
// later costs one field rather than one more argument at every call site.
//
// [Ja] commands は run が振り分ける先の実装。1 つずつ仮引数で受け取るのではなく 1 つの
// 値にまとめるのは、後から追加するサブコマンドが、すべての呼び出し箇所で増える引数では
// なく、1 つのフィールドで済むようにするため。
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

// run dispatches args to a subcommand and returns the process exit code. It
// takes the arguments, both output streams, and the subcommand implementations
// as parameters, rather than using the process's own values directly, so that
// the dispatch can be verified without terminating the test process, starting a
// server or emptying a database.
//
// Standard output is threaded through for devcreds, whose standard output is
// the machine-readable contract scripts/browse.sh reads. Reaching for os.Stdout
// inside the dispatch would put the one stream a caller parses out of a test's
// reach.
//
// No subcommand defaults to serve: an invocation that names nothing prints the
// usage and fails, so every call site has to state which subcommand it wants.
//
// [Ja] run は args をサブコマンドへ振り分け、プロセスの終了コードを返す。プロセス自身の値を
// 直接使わず、引数・2 つの出力先・サブコマンドの実装を仮引数で受け取るのは、テスト
// プロセスを終了させたり、サーバーを起動したり、データベースを空にしたりせずに振り分けを
// 検証できるようにするため。
//
// 標準出力を通しているのは devcreds のためで、その標準出力は scripts/browse.sh が読む
// 機械可読な契約になっている。振り分けの中で os.Stdout を直接掴むと、呼び出し側が解釈する
// 唯一のストリームがテストから触れなくなる。
//
// サブコマンド無しのときに serve へ既定することはしない。何も指定しない実行は usage を
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
		// The write error is discarded on purpose: this is the diagnostic
		// channel itself, so there is nowhere left to report a failure to write
		// to it. The exit code still tells the caller what happened.
		//
		// [Ja] 書き込みエラーは意図的に捨てる。ここは診断情報の出力先そのものであり、
		// その書き込みに失敗したことを報告する先が残っていないため。何が起きたかは
		// 終了コードで呼び出し側に伝わる。
		_, _ = fmt.Fprintf(stderr, "unknown subcommand: %q\n\n", args[0])
		usage(stderr)

		return exitUsage
	}
}

// runWithoutArguments runs a subcommand that takes none, and answers a command
// line that put something after it with the usage instead.
//
// The unexpected arguments are named back for the same reason an unknown
// subcommand is: a command line that only learns it was rejected cannot see
// which part of it was not understood. Neither of these subcommands takes an
// argument, and both of them look the same however they were invoked once they
// have started, so an ignored argument would leave a mistyped flag with no
// symptom at all: the server would come up on the default configuration, and
// the seed would empty the database, without a word.
//
// [Ja] runWithoutArguments は、引数を取らないサブコマンドを実行し、その後ろに何かを
// 続けたコマンドラインには、代わりに usage で応答する。
//
// 余分な引数を出力に含める理由は未知のサブコマンドと同じ。拒否されたことだけを知らされた
// コマンドラインからは、そのどの部分が解釈されなかったのかが分からない。ここに並ぶ
// サブコマンドはいずれも引数を取らず、走り出してしまえばどう起動されても見え方が同じで
// あるため、引数を無視すると打ち間違えたフラグには症状が 1 つも残らない。サーバーは既定の
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

// runWithOneArgument runs a subcommand that takes exactly one, and answers a
// command line that gave it none or several with the usage instead.
//
// What the argument is for is named back along with what was written, because
// this subcommand is reached from a shell script as often as from a terminal: a
// script that lost its variable passes an empty string, and one that lost its
// quoting passes several words, and neither of those reads as "the role is
// missing" from the command line alone.
//
// [Ja] runWithOneArgument は、引数をちょうど 1 つ取るサブコマンドを実行し、それを
// 与えなかった / 複数与えたコマンドラインには、代わりに usage で応答する。
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

// usage writes the list of available subcommands to w. The write error is
// discarded for the same reason as in run.
//
// [Ja] usage は利用可能なサブコマンドの一覧を w に書く。書き込みエラーを捨てる理由は
// run と同じ。
func usage(w io.Writer) {
	_, _ = fmt.Fprint(w, `usage: mewst <command>

commands:
  serve              start the HTTP server
  seed               rebuild the development database from the seed data
  devcreds <role>    print the email address and password of a seeded account
`)
}
