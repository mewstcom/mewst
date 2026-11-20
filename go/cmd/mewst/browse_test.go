package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// browseLoginHarnessはbrowse.shをsourceして関数を取り込み、プロセスの外へ
// 出るもの (Goツールチェイン、playwright-cli、ベースURLが要るBasic認証configの
// 生成) だけを差し替える。テスト対象 (役割選択、資格情報解析、生成されるパスワード
// スクリプト) は書かれたまま動く。
//
// ファイルパスはここに書き写さない。browse.shがMEWST_BROWSE_TMP_DIRから取るため、
// 定義はbrowse.shのものだけになり、あちらで改名したときに、このハーネスだけが別の
// 場所を指したまま、スクリプトが開発者の実tmpディレクトリへパスワードのファイルを
// 書く、という状態にならない。
const browseLoginHarness = `
set -euo pipefail

source "$MEWST_BROWSE_TEST_SCRIPT"

test_password="$MEWST_BROWSE_TEST_PASSWORD"
test_email="$MEWST_BROWSE_TEST_EMAIL"
test_credentials="${MEWST_BROWSE_TEST_CREDENTIALS:-}"
if [ -z "$test_credentials" ]; then
  printf -v test_credentials '%s\n%s' "$test_email" "$test_password"
fi
unset MEWST_BROWSE_TEST_PASSWORD MEWST_BROWSE_TEST_EMAIL MEWST_BROWSE_TEST_CREDENTIALS

go() {
  if [[ "$#" -eq 4 && "$1" == "run" && "$2" == "./cmd/mewst" && "$3" == "devcreds" ]]; then
    printf '%s' "$PWD" > "$MEWST_BROWSE_TEST_CAPTURE/go-dir"
    printf '%s' "$4" > "$MEWST_BROWSE_TEST_CAPTURE/role"
    if [ "$MEWST_BROWSE_TEST_DEVCREDS_FAILS" = "1" ]; then
      printf 'role rejected by devcreds\n' >&2
      return 7
    fi
    printf '%s' "$test_credentials"
    return 0
  fi

  printf 'unexpected go command:' >&2
  printf ' %q' "$@" >&2
  printf '
' >&2
  return 64
}

build_config() {
  mkdir -p "$TMP_DIR"
  printf '{}
' > "$CONFIG_FILE"
  chmod 600 "$CONFIG_FILE"
  printf '%s' 'https://example.test' > "$ORIGIN_FILE"
}

# pw は pw_checked ではなく playwright-cli 自体の代役。どちらのラッパーも書かれた
# まま動かし、ここで捕捉する argv を、実際のコマンドが渡されたはずの argv にするため。
#
# run-code には playwright-cli と同じ形で答える。実行したコードを返す形であり、
# サインインではそれが生成スクリプトと、その中のパスワードになる。これが
# pw_checked_secret の存在理由である応答そのものになる。実際の応答がその周りに置く
# js のコードフェンスは省いた。Go の raw string literal はバッククォートを持てないため。
# フィルタが手掛かりにするのは "### " のセクション見出しで、そちらは再現している。
pw() {
  local arg
  {
    printf 'playwright-cli'
    for arg in "$@"; do
      printf '	%s' "$arg"
    done
    printf '
'
  } >> "$MEWST_BROWSE_TEST_CAPTURE/playwright-argv"

  if [[ "$#" -ge 2 && "$1" == "run-code" && "$2" == --filename=* ]]; then
    local filename="${2#--filename=}"
    cp "$filename" "$MEWST_BROWSE_TEST_CAPTURE/password-script"
    stat -c '%a' "$filename" > "$MEWST_BROWSE_TEST_CAPTURE/password-mode"

    if [ "$MEWST_BROWSE_TEST_PASSWORD_FAILS" = "1" ]; then
      printf '### Error
TimeoutError: locator.fill: Timeout 30000ms exceeded.
'
    fi
    printf '### Ran Playwright code
await ('
    cat "$filename"
    printf ')(page);
'
  elif [[ "$#" -ge 2 && "$1" == "--raw" && "$2" == "run-code" ]]; then
    printf '"SIGNED_IN https://example.test/"\n'
  fi

  return 0
}

cmd_login "${1:-}"
`

// browseLoginPasswordは、シェル・JSON文字列・JavaScriptのソースがそれぞれ
// 特別扱いする文字を含む。受け渡しの各段がそれらに何をするのかを、仮定ではなく実際に
// 通すため。browseLoginEmailはdevcredsが1行目に返すアドレス。
const (
	browseLoginPassword = `p@ss word "$\quoted`
	browseLoginEmail    = "roster-user@example.com"
)

// browseLoginOptionsは、ハーネスの実行1回分の設定。2つのスイッチを位置では
// なく呼び出し側で名前付きにするのは、"false, true" がどちらの実行を求めているのかを
// 何も語らないため。
type browseLoginOptions struct {
	requestedRole string
	credentials   string
	devcredsFails bool
	passwordFails bool
}

type browseLoginResult struct {
	stdout string
	role   string
}

// TestBrowseLoginSelectsRoleは、既定アカウントと名簿の役割をそのまま指定する
// 経路を確認する。各経路でログインが使う資格情報解析も通す。
func TestBrowseLoginSelectsRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested string
		want      string
	}{
		{name: "指定が無いときはmain", requested: "", want: "main"},
		{name: "任意の役割はそのまま", requested: "follower", want: "follower"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := runBrowseLogin(t, tt.requested)

			if result.role != tt.want {
				t.Errorf("資格情報を役割%qで検索することを期待したが%qだった", tt.want, result.role)
			}
			if !strings.Contains(result.stdout, "logged in as "+tt.want+": https://example.test/") {
				t.Errorf("役割%qのログイン完了出力を期待したが%qだった", tt.want, result.stdout)
			}
		})
	}
}

// TestBrowseLoginRejectsMalformedCredentialsはdevcredsの厳密な2行契約を確認する。
func TestBrowseLoginRejectsMalformedCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		credentials string
	}{
		{name: "1行だけ", credentials: browseLoginEmail},
		{name: "3行", credentials: browseLoginEmail + "\n" + browseLoginPassword + "\nextra"},
		{name: "emailが空", credentials: "\n" + browseLoginPassword},
		{name: "末尾が改行だけの1行", credentials: browseLoginEmail + "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			command := browseLoginCommand(t, t.TempDir(), browseLoginOptions{
				requestedRole: "main",
				credentials:   tt.credentials,
			})
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr

			if err := command.Run(); err == nil {
				t.Fatalf("不正な資格情報では非ゼロ終了を期待したが成功した\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), "must print exactly two non-empty lines") {
				t.Errorf("2行契約のエラーを期待したが%qだった", stderr.String())
			}
			if strings.Contains(stdout.String(), browseLoginPassword) || strings.Contains(stderr.String(), browseLoginPassword) {
				t.Errorf("出力にパスワードが含まれている\nstdout: %q\nstderr: %q", stdout.String(), stderr.String())
			}
		})
	}
}

// TestBrowseLoginPreservesDevcredsFailureは子プロセスのエラーへ汎用エラーを
// 付け足さないことを確認する。
func TestBrowseLoginPreservesDevcredsFailure(t *testing.T) {
	t.Parallel()

	command := browseLoginCommand(t, t.TempDir(), browseLoginOptions{
		requestedRole: "missing",
		devcredsFails: true,
	})
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err == nil {
		t.Fatalf("devcreds失敗時は非ゼロ終了を期待したが成功した")
	}
	if stdout.Len() != 0 {
		t.Errorf("標準出力が空であることを期待したが%qだった", stdout.String())
	}
	if stderr.String() != "role rejected by devcreds\n" {
		t.Errorf("devcreds自身のstderrだけを期待したが%qだった", stderr.String())
	}
}

// TestBrowseLoginMakeTargetQuotesRoleはMake変数内のシェルメタ文字が引用された
// 1つのスクリプト引数に留まることを確認する。
func TestBrowseLoginMakeTargetQuotesRole(t *testing.T) {
	t.Parallel()

	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Goモジュールルートの絶対パスを取得できなかった: %v", err)
	}
	command := exec.CommandContext(t.Context(), "make", "-n", "browse-login", "BROWSE_USER=main; printf injected")
	command.Dir = moduleRoot
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("browse-loginのdry-runに失敗: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), `login "main; printf injected"`) {
		t.Errorf("役割が引用されたコマンドを期待したが%qだった", output)
	}
}

// TestBrowseLoginKeepsPasswordOutOfFailureOutputは、パスワード入力の手順が
// 失敗したときを扱う。playwright-cliはrun-codeへ、実行したコードを返す。ここでは
// それが生成スクリプトと、その中のパスワードになるため、応答の本文は、失敗時に
// 出してはならない唯一のものになる。
//
// 実行は失敗してそう告げ、Errorセクションは失敗を調査できるようにするため標準エラー
// 出力へ届き、応答のそれ以外は届かない。
func TestBrowseLoginKeepsPasswordOutOfFailureOutput(t *testing.T) {
	t.Parallel()

	captureDir := t.TempDir()

	command := browseLoginCommand(t, captureDir, browseLoginOptions{requestedRole: "main", passwordFails: true})
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err == nil {
		t.Fatalf("パスワード入力に失敗したときは非ゼロ終了を期待したが成功した\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}

	if strings.Contains(stderr.String(), browseLoginPassword) {
		t.Errorf("標準エラー出力にパスワードが含まれている: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), browseLoginPassword) {
		t.Errorf("標準出力にパスワードが含まれている: %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "Ran Playwright code") {
		t.Errorf("実行したコードのセクションを出さないことを期待したが%qだった", stderr.String())
	}
	if !strings.Contains(stderr.String(), "TimeoutError: locator.fill") {
		t.Errorf("Errorセクションの内容を出すことを期待したが%qだった", stderr.String())
	}
	if !strings.Contains(stderr.String(), "filling the password failed") {
		t.Errorf("パスワード入力の失敗を告げる出力を期待したが%qだった", stderr.String())
	}

	// 明示的な削除は失敗した手順の後ろにあるため、抜ける途中でtrapが
	// スクリプトを持って行く必要がある。
	if _, err := os.Stat(filepath.Join(browseTmpDir(captureDir), "browse-cli.password.js")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("失敗時もパスワードスクリプトが削除されることを期待したがerr=%v", err)
	}
}

// browseLoginCommandはハーネスの実行を組み立てる。captureDirはスタブが記録
// するものと、browse.shを向けるtmpディレクトリの両方を持つため、テストは自身の
// t.TempDirの外に触れない。
func browseLoginCommand(t *testing.T, captureDir string, opts browseLoginOptions) *exec.Cmd {
	t.Helper()

	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "browse.sh"))
	if err != nil {
		t.Fatalf("browse.shの絶対パスを取得できなかった: %v", err)
	}

	args := []string{"-c", browseLoginHarness, "browse-test"}
	if opts.requestedRole != "" {
		args = append(args, opts.requestedRole)
	}

	command := exec.CommandContext(t.Context(), "bash", args...)
	command.Dir = captureDir
	command.Env = append(
		os.Environ(),
		"MEWST_BROWSE_TEST_SCRIPT="+scriptPath,
		"MEWST_BROWSE_TEST_CAPTURE="+captureDir,
		"MEWST_BROWSE_TEST_PASSWORD="+browseLoginPassword,
		"MEWST_BROWSE_TEST_EMAIL="+browseLoginEmail,
		"MEWST_BROWSE_TEST_CREDENTIALS="+opts.credentials,
		"MEWST_BROWSE_TEST_DEVCREDS_FAILS="+boolFlag(opts.devcredsFails),
		"MEWST_BROWSE_TEST_PASSWORD_FAILS="+boolFlag(opts.passwordFails),
		"MEWST_BROWSE_TMP_DIR="+browseTmpDir(captureDir),
	)

	return command
}

// browseTmpDirは、テスト中にbrowse.shが資格情報を含むファイルを書く場所。
// スクリプトから読み取るのではなくここで指定する。どこへ書くのかをスクリプトに
// 尋ねるテストは、誤った答えと正しい答えを区別できないため。
func browseTmpDir(captureDir string) string {
	return filepath.Join(captureDir, "tmp")
}

// boolFlagは、ハーネスが文字列比較で読むスイッチを組み立てる。
func boolFlag(on bool) string {
	if on {
		return "1"
	}

	return "0"
}

func runBrowseLogin(t *testing.T, requestedRole string) browseLoginResult {
	t.Helper()

	captureDir := t.TempDir()

	command := browseLoginCommand(t, captureDir, browseLoginOptions{requestedRole: requestedRole})
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("browse.shのログインテストに失敗: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("標準エラー出力が空であることを期待したが%qだった", stderr.String())
	}

	readCapture := func(name string) string {
		t.Helper()

		body, err := os.ReadFile(filepath.Join(captureDir, name))
		if err != nil {
			t.Fatalf("%sを読み込めなかった: %v", name, err)
		}

		return string(body)
	}

	wantGoDir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Goモジュールルートの絶対パスを取得できなかった: %v", err)
	}
	if goDir := readCapture("go-dir"); goDir != wantGoDir {
		t.Errorf("モジュール外からでもdevcredsを%qで実行することを期待したが%qだった", wantGoDir, goDir)
	}

	calls := readCapture("playwright-argv")
	if strings.Contains(calls, browseLoginPassword) {
		t.Errorf("playwright-cliのargvにパスワードが含まれている: %q", calls)
	}
	if !strings.Contains(calls, "\tfill\tinput[name=\"email\"]\t"+browseLoginEmail+"\n") {
		t.Errorf("devcredsの1行目をemail入力に使うことを期待したが%qだった", calls)
	}

	passwordScript := readCapture("password-script")
	quotedPassword, err := json.Marshal(browseLoginPassword)
	if err != nil {
		t.Fatalf("パスワードをJSON文字列へ変換できなかった: %v", err)
	}
	if !strings.Contains(passwordScript, string(quotedPassword)) {
		t.Errorf("devcredsの2行目をパスワード入力に使うことを期待したが%qだった", passwordScript)
	}
	if mode := strings.TrimSpace(readCapture("password-mode")); mode != "600" {
		t.Errorf("パスワードスクリプトの権限が600であることを期待したが%qだった", mode)
	}
	// playwright-cliへ渡されたパスは、このテストがbrowse.shを向けた先で
	// ある必要がある。これが無いと、下の削除の確認は、まったく別の場所へスクリプトを
	// 書いてそのまま残した実行に対しても成立してしまう。
	if want := "\t--filename=" + filepath.Join(browseTmpDir(captureDir), "browse-cli.password.js") + "\n"; !strings.Contains(calls, want) {
		t.Errorf("パスワードスクリプトを%qへ書くことを期待したが%qだった", want, calls)
	}
	if _, err := os.Stat(filepath.Join(browseTmpDir(captureDir), "browse-cli.password.js")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ログイン後にパスワードスクリプトが削除されることを期待したがerr=%v", err)
	}

	return browseLoginResult{
		stdout: stdout.String(),
		role:   readCapture("role"),
	}
}
