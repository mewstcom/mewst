package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/mewstcom/mewst/go/internal/seed"
)

// runDevCredentialsは、名簿がroleに与えているシード済みアカウントの資格
// 情報をstdoutへ書き出す。
//
// 出力先を自分で読まずに仮引数で受け取るのは、振り分けから、呼び出し側が読む
// 2行までを検証できるようにするため。
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

// writeDevCredentialsは、アドレスとパスワードを1行ずつ、それだけをwへ
// 書く。
//
// 呼び出し側はシェルスクリプトであり、この2行を変数へ読み込む。ラベル・引用符・
// 区切り文字はいずれも、呼び出し側が剥がし直さなければならないものであり、その
// 剥がし方を誤ることが、パスワードが1文字多い / 少ない状態でサインインフォームへ
// 届く経路になる。パスワードを引数で渡さずここへ書くのは、引数から読まないのと
// 同じ理由による。argvはマシン上のすべてのプロセスから見える。
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
