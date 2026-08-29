package main

import (
	"bytes"
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
		serve: func() { t.Error("serve が呼ばれた") },
		seed:  func() { t.Error("seed が呼ばれた") },
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

			code := run(tt.args, &stderr, noCommands(t))

			if code != exitUsage {
				t.Errorf("run() exit code = %d, want %d", code, exitUsage)
			}
			for _, want := range []string{
				"usage: mewst <command>",
				"  serve   start the HTTP server",
				"  seed    rebuild the development database from the seed data",
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

	run([]string{"nosuchcommand"}, &stderr, noCommands(t))

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

			code := run([]string{tt.arg}, &stderr, commands{
				serve: func() { calls["serve"]++ },
				seed:  func() { calls["seed"]++ },
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

			code := run(tt.args, &stderr, noCommands(t))

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
