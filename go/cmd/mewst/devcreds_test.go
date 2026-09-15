package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/mewstcom/mewst/go/internal/seed"
)

// TestWriteDevCredentialsは出力の形を固定する。呼び出し側は、アドレスと
// パスワードを2つの変数へ読み込むシェルスクリプトであり、この2行と、そこに
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
		t.Errorf("writeDevCredentials()の標準出力 = %q、期待値 = %q", stdout.String(), want)
	}
}

// failingWriterは、居なくなった呼び出し側の標準出力がそうであるように、
// すべての書き込みに失敗する。
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

// errWriteFailedは、標準出力の向こう側が居なくなったときにOSが報告する
// ものを代表する。
var errWriteFailed = errors.New("書き込み先が閉じられています")

// TestWriteDevCredentialsReportsAFailedWriteは、届かなかった書き込みが、
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
