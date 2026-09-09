package main

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"
)

// noCommands is the set of implementations for a dispatch that is expected to
// reach none of them. Calling one is the failure, so each records that it was
// called rather than doing anything.
//
// [Ja] noCommands は、どの実装にも辿り着かないことを期待する振り分けのための実装の
// 組。呼ばれること自体が失敗であるため、それぞれは何かを行うのではなく、呼ばれた
// ことを記録する。
func noCommands(t *testing.T) commands {
	t.Helper()

	return commands{
		serve:    func() { t.Error("serve が呼ばれた") },
		seed:     func() { t.Error("seed が呼ばれた") },
		devcreds: func(_ io.Writer, role string) { t.Errorf("devcreds が役割 %q で呼ばれた", role) },
	}
}

// TestRun_RejectsAnInvocationWithoutAKnownSubcommand verifies that a command
// line naming no subcommand, or one this command does not know, is answered
// with the usage and the usage exit code.
//
// [Ja] TestRun_RejectsAnInvocationWithoutAKnownSubcommand は、サブコマンドを指定して
// いない / 本コマンドの知らないサブコマンドを指定したコマンドラインが、usage と使用方法の
// 誤りを示す終了コードで応答されることを検証する。
func TestRun_RejectsAnInvocationWithoutAKnownSubcommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "サブコマンドを指定していない"},
		{name: "未知のサブコマンドを指定した", args: []string{"nosuchcommand"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer

			code := run(tt.args, io.Discard, &stderr, noCommands(t))

			if code != exitUsage {
				t.Errorf("run() exit code = %d, want %d", code, exitUsage)
			}
			for _, want := range []string{
				"usage: mewst <command>",
				"  serve              start the HTTP server",
				"  seed               rebuild the development database from the seed data",
				"  devcreds <role>    print the email address and password of a seeded account",
			} {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("run() stderr = %q, want it to contain %q", stderr.String(), want)
				}
			}
		})
	}
}

// TestRun_NamesTheUnknownSubcommand pins the quoting of the name that was not
// understood. Without it the output tells the reader only that some name was
// rejected, which is the one thing they already know.
//
// [Ja] TestRun_NamesTheUnknownSubcommand は、解釈できなかった名前を引用符付きで
// 出力することを固定する。これが無いと、出力は「何らかの名前が拒否された」ことしか
// 伝えず、それは読み手がすでに知っている唯一のことになる。
func TestRun_NamesTheUnknownSubcommand(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer

	run([]string{"nosuchcommand"}, io.Discard, &stderr, noCommands(t))

	if want := `unknown subcommand: "nosuchcommand"`; !strings.Contains(stderr.String(), want) {
		t.Errorf("run() stderr = %q, want it to contain %q", stderr.String(), want)
	}
}

// TestRun_DispatchesToTheNamedSubcommand verifies that each name reaches its
// own implementation, and only its own, without starting the blocking HTTP
// server or emptying a database.
//
// [Ja] TestRun_DispatchesToTheNamedSubcommand は、それぞれの名前が自身の実装に、
// かつ自身の実装だけに辿り着くことを、ブロックする HTTP サーバーを起動したり
// データベースを空にしたりせずに検証する。
func TestRun_DispatchesToTheNamedSubcommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  string
	}{
		{name: "serve", arg: "serve"},
		{name: "seed", arg: "seed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			calls := map[string]int{}

			code := run([]string{tt.arg}, io.Discard, &stderr, commands{
				serve:    func() { calls["serve"]++ },
				seed:     func() { calls["seed"]++ },
				devcreds: func(io.Writer, string) { calls["devcreds"]++ },
			})

			if code != 0 {
				t.Errorf("run() exit code = %d, want 0", code)
			}
			if calls[tt.arg] != 1 {
				t.Errorf("%s calls = %d, want 1", tt.arg, calls[tt.arg])
			}
			if len(calls) != 1 {
				t.Errorf("run() reached %v, want only %s", calls, tt.arg)
			}
			if stderr.Len() != 0 {
				t.Errorf("run() stderr = %q, want empty", stderr.String())
			}
		})
	}
}

// TestRun_RejectsArgumentsAfterASubcommand verifies that a command line that
// puts anything after a subcommand is answered with the usage instead of the
// work. Neither of these subcommands takes an argument, and both look the same
// however they were invoked once they have started, so an ignored argument
// would leave a mistyped flag with no symptom at all.
//
// [Ja] TestRun_RejectsArgumentsAfterASubcommand は、サブコマンドの後ろに何かを続けた
// コマンドラインが、その処理ではなく usage で応答されることを検証する。ここに並ぶ
// サブコマンドはいずれも引数を取らず、走り出してしまえばどう起動されても見え方が同じで
// あるため、引数を無視すると打ち間違えたフラグには症状が 1 つも残らない。
func TestRun_RejectsArgumentsAfterASubcommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{
			name:    "serve",
			args:    []string{"serve", "--port", "3000"},
			wantMsg: `serve takes no arguments: ["--port" "3000"]`,
		},
		{
			name:    "seed",
			args:    []string{"seed", "--force"},
			wantMsg: `seed takes no arguments: ["--force"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer

			code := run(tt.args, io.Discard, &stderr, noCommands(t))

			if code != exitUsage {
				t.Errorf("run() exit code = %d, want %d", code, exitUsage)
			}
			for _, want := range []string{tt.wantMsg, "usage: mewst <command>"} {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("run() stderr = %q, want it to contain %q", stderr.String(), want)
				}
			}
		})
	}
}

// TestRun_DispatchesTheRoleToDevcreds verifies that the role written on the
// command line reaches the subcommand as it was written, and that what the
// subcommand writes reaches the standard output the caller supplied. The role
// is the only argument any subcommand takes, and it decides which account's
// password is printed; that standard output is the stream the caller reads the
// password from.
//
// [Ja] TestRun_DispatchesTheRoleToDevcreds は、コマンドラインに書かれた役割が
// 書かれたままサブコマンドへ届くこと、そしてサブコマンドが書いたものが、呼び出し側の
// 渡した標準出力へ届くことを検証する。役割はどのサブコマンドを通しても唯一の引数で
// あり、どのアカウントのパスワードを出力するのかを決めるものになる。その標準出力は、
// 呼び出し側がパスワードを読む先のストリームである。
func TestRun_DispatchesTheRoleToDevcreds(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	var got []string

	code := run([]string{"devcreds", "follower"}, &stdout, &stderr, commands{
		serve: func() { t.Error("serve が呼ばれた") },
		seed:  func() { t.Error("seed が呼ばれた") },
		devcreds: func(w io.Writer, role string) {
			got = append(got, role)
			if _, err := io.WriteString(w, "seeduser2@example.com\nseed-password\n"); err != nil {
				t.Errorf("資格情報の書き込みに失敗: %v", err)
			}
		},
	})

	if code != 0 {
		t.Errorf("run() exit code = %d, want 0", code)
	}
	if want := []string{"follower"}; !slices.Equal(got, want) {
		t.Errorf("devcreds calls = %v, want %v", got, want)
	}
	if want := "seeduser2@example.com\nseed-password\n"; stdout.String() != want {
		t.Errorf("run() stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Errorf("run() stderr = %q, want empty", stderr.String())
	}
}

// TestRun_RejectsDevcredsWithoutExactlyOneRole verifies that a command line
// that named no role, or more than one, is answered with the usage. A role it
// does not receive is one the subcommand would otherwise have to guess at,
// and the account it guessed would be the one whose password is printed.
//
// The standard output has to stay empty in that case: the caller reads that
// stream as the credentials themselves, so a rejected invocation says what
// happened on standard error and leaves nothing to be read as a password.
//
// [Ja] TestRun_RejectsDevcredsWithoutExactlyOneRole は、役割を指定していない /
// 2 つ以上指定したコマンドラインが usage で応答されることを検証する。受け取れな
// かった役割は、サブコマンドが推測するほかないものであり、推測されたアカウントは
// そのパスワードが出力されるアカウントになる。
//
// このとき標準出力は空のままである必要がある。呼び出し側はそのストリームを資格情報
// そのものとして読むため、拒否された実行は、何が起きたのかを標準エラー出力で告げ、
// パスワードとして読まれるものを残さない。
func TestRun_RejectsDevcredsWithoutExactlyOneRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{
			name:    "役割を指定していない",
			args:    []string{"devcreds"},
			wantMsg: `devcreds takes exactly one argument <role>: []`,
		},
		{
			name:    "役割を 2 つ指定した",
			args:    []string{"devcreds", "main", "follower"},
			wantMsg: `devcreds takes exactly one argument <role>: ["main" "follower"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr, noCommands(t))

			if code != exitUsage {
				t.Errorf("run() exit code = %d, want %d", code, exitUsage)
			}
			for _, want := range []string{tt.wantMsg, "usage: mewst <command>"} {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("run() stderr = %q, want it to contain %q", stderr.String(), want)
				}
			}
			if stdout.Len() != 0 {
				t.Errorf("run() stdout = %q, want empty", stdout.String())
			}
		})
	}
}
