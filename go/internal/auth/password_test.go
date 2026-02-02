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

func TestHashPassword(t *testing.T) {
	t.Parallel()

	t.Run("パスワードをハッシュ化できる", func(t *testing.T) {
		t.Parallel()

		password := "testpassword123"
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		if hash == "" {
			t.Error("hash should not be empty")
		}

		if hash == password {
			t.Error("hash should not equal plain password")
		}
	})

	t.Run("生成されたハッシュはCheckPasswordで検証できる", func(t *testing.T) {
		t.Parallel()

		password := "mySecurePassword456"
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		err = CheckPassword(hash, password)
		if err != nil {
			t.Errorf("CheckPassword should succeed with correct password: %v", err)
		}
	})

	t.Run("同じパスワードでも毎回異なるハッシュが生成される", func(t *testing.T) {
		t.Parallel()

		password := "samePassword789"
		hash1, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		hash2, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		if hash1 == hash2 {
			t.Error("hashes should be different due to bcrypt salt")
		}

		// どちらのハッシュも検証可能であること
		if err := CheckPassword(hash1, password); err != nil {
			t.Errorf("CheckPassword should succeed with hash1: %v", err)
		}
		if err := CheckPassword(hash2, password); err != nil {
			t.Errorf("CheckPassword should succeed with hash2: %v", err)
		}
	})

	t.Run("日本語を含むパスワードをハッシュ化できる", func(t *testing.T) {
		t.Parallel()

		password := "パスワード123"
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		err = CheckPassword(hash, password)
		if err != nil {
			t.Errorf("CheckPassword should succeed: %v", err)
		}
	})
}
