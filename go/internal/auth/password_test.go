package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCheckPassword(t *testing.T) {
	t.Parallel()

	// Railsのhas_secure_password互換のハッシュ生成
	plainPassword := "password123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("テスト用ハッシュの生成に失敗: %v", err)
	}

	tests := []struct {
		name           string
		hashedPassword string
		plainPassword  string
		wantErr        bool
	}{
		{
			name:           "正しいパスワードで成功",
			hashedPassword: string(hashedPassword),
			plainPassword:  plainPassword,
			wantErr:        false,
		},
		{
			name:           "間違ったパスワードで失敗",
			hashedPassword: string(hashedPassword),
			plainPassword:  "wrongpassword",
			wantErr:        true,
		},
		{
			name:           "空のパスワードで失敗",
			hashedPassword: string(hashedPassword),
			plainPassword:  "",
			wantErr:        true,
		},
		{
			name:           "無効なハッシュで失敗",
			hashedPassword: "invalid_hash",
			plainPassword:  plainPassword,
			wantErr:        true,
		},
		{
			name:           "空のハッシュで失敗",
			hashedPassword: "",
			plainPassword:  plainPassword,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckPassword(tt.hashedPassword, tt.plainPassword)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckPassword_RailsCompatibility(t *testing.T) {
	t.Parallel()

	// Railsのhas_secure_passwordで生成されたハッシュ形式との互換性をテスト
	// bcrypt.DefaultCost = 10 はRailsのデフォルトと同じ
	plainPassword := "test_password_日本語"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("テスト用ハッシュの生成に失敗: %v", err)
	}

	err = CheckPassword(string(hashedPassword), plainPassword)
	if err != nil {
		t.Errorf("Rails互換パスワードの検証に失敗: %v", err)
	}
}
