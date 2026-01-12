// Package auth は認証機能を提供します
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword はパスワードをbcryptでハッシュ化する
// Rails版のhas_secure_passwordとの互換性を保つためコスト12を使用
func HashPassword(plainPassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
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
