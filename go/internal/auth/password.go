// Package auth は認証機能を提供します
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// BcryptCost はbcryptのコスト値。テスト時はTestBcryptCostに変更される
var BcryptCost = bcrypt.DefaultCost

// TestBcryptCost はテスト用の低コスト値
const TestBcryptCost = bcrypt.MinCost

// HashPassword はパスワードをbcryptでハッシュ化する
func HashPassword(plainPassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword bcryptでハッシュ化されたパスワードと平文パスワードを比較する
// Rails版のhas_secure_passwordで生成されたpassword_digestカラムとの互換性を保つ
func CheckPassword(hashedPassword, plainPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
}
