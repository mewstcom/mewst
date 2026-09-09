package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/mewstcom/mewst/go/internal/seed"
)

// runDevCredentials writes the credentials of the seeded account the roster
// gives role to stdout.
//
// The stream is taken as a parameter rather than read here, so that the
// dispatch can be verified all the way to the two lines a caller reads.
//
// [Ja] runDevCredentials は、名簿が role に与えているシード済みアカウントの資格
// 情報を stdout へ書き出す。
//
// 出力先を自分で読まずに仮引数で受け取るのは、振り分けから、呼び出し側が読む
// 2 行までを検証できるようにするため。
func runDevCredentials(stdout io.Writer, role string) {
	credentials, err := seed.DevCredentials(role)
	if err != nil {
		slog.Error("開発用アカウントの資格情報の取得に失敗しました", "error", err)
		os.Exit(1)
	}

	if err := writeDevCredentials(stdout, credentials); err != nil {
		slog.Error("開発用アカウントの資格情報の出力に失敗しました", "error", err)
		os.Exit(1)
	}
}

// writeDevCredentials writes the address and the password to w, one per line
// and nothing else.
//
// The caller is a shell script, which reads the two lines into variables. A
// label, a quote or a separator would each be something it had to strip back
// off, and stripping it wrongly is how a password reaches the sign-in form
// with a character too many or too few. The password is written here rather
// than passed as an argument for the same reason it is not read from one:
// argv is visible to every process on the machine.
//
// The write error is returned rather than discarded, unlike the ones the
// progress report drops. This stream carries the value the caller asked for,
// so a caller that received only half of it has to be told, not left to sign
// in with a truncated password.
//
// [Ja] writeDevCredentials は、アドレスとパスワードを 1 行ずつ、それだけを w へ
// 書く。
//
// 呼び出し側はシェルスクリプトであり、この 2 行を変数へ読み込む。ラベル・引用符・
// 区切り文字はいずれも、呼び出し側が剥がし直さなければならないものであり、その
// 剥がし方を誤ることが、パスワードが 1 文字多い / 少ない状態でサインインフォームへ
// 届く経路になる。パスワードを引数で渡さずここへ書くのは、引数から読まないのと
// 同じ理由による。argv はマシン上のすべてのプロセスから見える。
//
// 書き込みエラーは、進捗の報告が捨てているものとは違って返す。このストリームは
// 呼び出し側が求めた値そのものを運んでおり、その半分しか受け取らなかった呼び出し
// 側は、切り詰められたパスワードでサインインするに任せるのではなく、そう告げられる
// 必要があるため。
func writeDevCredentials(w io.Writer, credentials seed.Credentials) error {
	if _, err := fmt.Fprintf(w, "%s\n%s\n", credentials.Email, credentials.Password); err != nil {
		return fmt.Errorf("資格情報の書き込みに失敗: %w", err)
	}

	return nil
}
