// Command mewst is Mewst's command-line entry point. The serve subcommand
// starts the HTTP server, and the seed subcommand rebuilds a development
// database from the seed data.
//
// The binary is named after the service rather than after the one role it
// started out filling, so that the operational one-off tasks that follow are
// added as subcommands instead of as a binary each.
//
// [Ja] mewst コマンドは Mewst のコマンドラインエントリーポイント。serve サブコマンドが
// HTTP サーバーを起動し、seed サブコマンドが開発用データベースをシードデータで作り直す。
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
	serve func()
	seed  func()
}

func main() {
	os.Exit(run(os.Args[1:], os.Stderr, commands{serve: runServe, seed: runSeed}))
}

// run dispatches args to a subcommand and returns the process exit code. It
// takes the arguments, the diagnostic stream, and the subcommand
// implementations as parameters, rather than using the process's own values
// directly, so that the dispatch can be verified without terminating the test
// process, starting a server or emptying a database.
//
// No subcommand defaults to serve: an invocation that names nothing prints the
// usage and fails, so every call site has to state which subcommand it wants.
//
// [Ja] run は args をサブコマンドへ振り分け、プロセスの終了コードを返す。プロセス自身の値を
// 直接使わず、引数・診断情報の出力先・サブコマンドの実装を仮引数で受け取るのは、テスト
// プロセスを終了させたり、サーバーを起動したり、データベースを空にしたりせずに振り分けを
// 検証できるようにするため。
//
// サブコマンド無しのときに serve へ既定することはしない。何も指定しない実行は usage を
// 表示して失敗するため、各呼び出し箇所がどのサブコマンドを使うのかを明示することになる。
func run(args []string, stderr io.Writer, cmds commands) int {
	if len(args) == 0 {
		usage(stderr)

		return exitUsage
	}

	switch args[0] {
	case "serve":
		return runWithoutArguments("serve", args[1:], stderr, cmds.serve)
	case "seed":
		return runWithoutArguments("seed", args[1:], stderr, cmds.seed)
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

// usage writes the list of available subcommands to w. The write error is
// discarded for the same reason as in run.
//
// [Ja] usage は利用可能なサブコマンドの一覧を w に書く。書き込みエラーを捨てる理由は
// run と同じ。
func usage(w io.Writer) {
	_, _ = fmt.Fprint(w, `usage: mewst <command>

commands:
  serve   start the HTTP server
  seed    rebuild the development database from the seed data
`)
}
