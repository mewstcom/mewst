// Package auth は認証機能を提供します
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// CheckPassword bcryptでハッシュ化されたパスワードと平文パスワードを比較する
// Rails版のhas_secure_passwordで生成されたpassword_digestカラムとの互換性を保つ
func CheckPassword(hashedPassword, plainPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
}
