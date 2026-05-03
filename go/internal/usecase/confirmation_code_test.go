package usecase

import (
	"regexp"
	"testing"
)

func TestGenerateConfirmationCode(t *testing.T) {
	t.Parallel()

	t.Run("6桁の数字コードが生成される", func(t *testing.T) {
		t.Parallel()

		code, err := generateConfirmationCode()
		if err != nil {
			t.Fatalf("generateConfirmationCode() error = %v", err)
		}

		if len(code) != 6 {
			t.Errorf("コード長 = %d, want 6", len(code))
		}

		matched, _ := regexp.MatchString(`^[0-9]{6}$`, code)
		if !matched {
			t.Errorf("コードが6桁の数字ではありません: %q", code)
		}
	})

	t.Run("生成されるコードはランダムである", func(t *testing.T) {
		t.Parallel()

		codes := make(map[string]bool)
		for i := 0; i < 100; i++ {
			code, err := generateConfirmationCode()
			if err != nil {
				t.Fatalf("generateConfirmationCode() error = %v", err)
			}
			codes[code] = true
		}

		// 100回生成して、少なくとも90種類以上の異なるコードが生成されることを確認
		if len(codes) < 90 {
			t.Errorf("ユニークなコード数 = %d, want >= 90", len(codes))
		}
	})
}
