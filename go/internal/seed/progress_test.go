package seed

import (
	"bytes"
	"strings"
	"testing"
)

// TestProgress_Accounts verifies that the report a run ends with carries, for
// every account, the four things an account is picked by.
//
// [Ja] TestProgress_Accounts は、実行の最後の報告が、すべてのアカウントについて、
// アカウントを選ぶための 4 つを持つことを検証する。
func TestProgress_Accounts(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	newProgress(&out).accounts([]seedAccount{
		{
			roster: rosterUser{
				role:   roleMain,
				atname: "seeduser1",
				email:  "seeduser1@example.com",
				note:   "主な確認対象",
			},
		},
		{
			roster: rosterUser{
				role:   roleDiscarded,
				atname: "seeduser5",
				email:  "seeduser5@example.com",
				note:   "削除済みプロフィール",
			},
		},
	})

	for _, want := range []string{
		"main", "seeduser1", "seeduser1@example.com", "主な確認対象",
		"discarded", "seeduser5", "seeduser5@example.com", "削除済みプロフィール",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("報告が %q を含むことを期待したが、出力は %q だった", want, out.String())
		}
	}

	// One account per line is what makes the report readable down a column,
	// which is how it is read when one account out of five is being picked.
	//
	// [Ja] 1 行に 1 アカウントであることが、報告を列に沿って読めるものにする。
	// 5 件から 1 件を選ぶときに読まれるのがその読み方であるため。
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if strings.Contains(line, "seeduser1") && strings.Contains(line, "seeduser5") {
			t.Errorf("1 行に複数のアカウントが並んでいる: %q", line)
		}
	}
}

// TestProgress_Line verifies that a line goes out whole and on its own, since
// it shares a stream with the log lines around it.
//
// [Ja] TestProgress_Line は、1 行がそのまま 1 行として出ることを検証する。前後の
// ログ行とストリームを共有するため。
func TestProgress_Line(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	p := newProgress(&out)
	p.line("データベース %s を空にします", "mewst_dev")
	p.line("名簿は %s です", "seed-users.toml")

	want := "データベース mewst_dev を空にします\n名簿は seed-users.toml です\n"
	if out.String() != want {
		t.Errorf("出力が %q であることを期待したが %q だった", want, out.String())
	}
}
