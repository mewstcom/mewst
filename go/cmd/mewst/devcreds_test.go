package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/mewstcom/mewst/go/internal/seed"
)

// TestWriteDevCredentials pins the shape of the output. The caller is a shell
// script that reads the address and the password into two variables, so the
// two lines and the absence of anything else on them are the interface.
//
// [Ja] TestWriteDevCredentials は出力の形を固定する。呼び出し側は、アドレスと
// パスワードを 2 つの変数へ読み込むシェルスクリプトであり、この 2 行と、そこに
// 他の何も無いことがインターフェースになる。
func TestWriteDevCredentials(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	err := writeDevCredentials(&stdout, seed.Credentials{
		Email:    "seeduser1@example.com",
		Password: "seed password",
	})
	if err != nil {
		t.Fatalf("資格情報の出力に失敗: %v", err)
	}

	if want := "seeduser1@example.com\nseed password\n"; stdout.String() != want {
		t.Errorf("writeDevCredentials() stdout = %q, want %q", stdout.String(), want)
	}
}

// failingWriter fails every write, as the standard output of a caller that
// went away does.
//
// [Ja] failingWriter は、居なくなった呼び出し側の標準出力がそうであるように、
// すべての書き込みに失敗する。
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

// errWriteFailed stands for whatever the operating system reports when the
// other end of the standard output is gone.
//
// [Ja] errWriteFailed は、標準出力の向こう側が居なくなったときに OS が報告する
// ものを代表する。
var errWriteFailed = errors.New("書き込み先が閉じられています")

// TestWriteDevCredentialsReportsAFailedWrite verifies that a write that did
// not land is reported rather than dropped. A caller that received half of a
// password would otherwise go on to sign in with it.
//
// [Ja] TestWriteDevCredentialsReportsAFailedWrite は、届かなかった書き込みが、
// 捨てられるのではなく報告されることを検証する。そうしないと、パスワードの半分を
// 受け取った呼び出し側が、そのままそれでサインインしに行く。
func TestWriteDevCredentialsReportsAFailedWrite(t *testing.T) {
	t.Parallel()

	err := writeDevCredentials(failingWriter{}, seed.Credentials{
		Email:    "seeduser1@example.com",
		Password: "seed password",
	})
	if err == nil {
		t.Fatal("書き込みの失敗が報告されなかった")
	}
}
