package usecase

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// generateConfirmationCodeは6桁のランダムな数字コードを生成する
func generateConfirmationCode() (string, error) {
	// 000000から999999までのランダムな数を生成
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	// 6桁になるようにゼロ埋め
	return fmt.Sprintf("%06d", n.Int64()), nil
}
