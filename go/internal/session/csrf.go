package session

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateCSRFTokenは安全なCSRFトークンを生成する
// 32バイトのランダムデータをBase64エンコードして返す
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
