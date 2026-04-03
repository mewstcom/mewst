package auth

import (
	"testing"
)

func TestGenerateSecureToken(t *testing.T) {
	t.Parallel()

	token, err := GenerateSecureToken()
	if err != nil {
		t.Fatalf("トークン生成に失敗: %v", err)
	}

	// Base64 URL-safe エンコードされた24バイト = 32文字
	if len(token) != 32 {
		t.Errorf("トークンの長さが不正: got %d, want 32", len(token))
	}

	// 2回生成して異なることを確認
	token2, err := GenerateSecureToken()
	if err != nil {
		t.Fatalf("2回目のトークン生成に失敗: %v", err)
	}

	if token == token2 {
		t.Error("2回生成したトークンが同じ値になってしまった")
	}
}
