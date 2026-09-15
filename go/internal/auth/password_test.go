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
				t.Errorf("CheckPassword()のエラー = %v、エラーの有無の期待値 = %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckPassword_RailsCompatibility(t *testing.T) {
	t.Parallel()

	// Railsのhas_secure_passwordで生成されたハッシュ形式との互換性をテスト
	// bcrypt.DefaultCost = 10はRailsのデフォルトと同じ
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

func TestBcryptCostValues(t *testing.T) {
	t.Parallel()

	t.Run("デフォルトのコストはbcrypt.DefaultCost", func(t *testing.T) {
		t.Parallel()

		if BcryptCost != bcrypt.DefaultCost {
			t.Errorf("BcryptCost = %d、期待値 = %d", BcryptCost, bcrypt.DefaultCost)
		}
	})

	t.Run("TestBcryptCostはbcrypt.MinCost", func(t *testing.T) {
		t.Parallel()

		if TestBcryptCost != bcrypt.MinCost {
			t.Errorf("TestBcryptCost = %d、期待値 = %d", TestBcryptCost, bcrypt.MinCost)
		}
	})
}

func TestHashPassword(t *testing.T) {
	t.Parallel()

	t.Run("パスワードをハッシュ化できる", func(t *testing.T) {
		t.Parallel()

		password := "testpassword123"
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword()のエラー = %v", err)
		}

		if hash == "" {
			t.Error("ハッシュ = 空文字列、非空を期待")
		}

		if hash == password {
			t.Error("ハッシュが平文のパスワードと一致している")
		}
	})

	t.Run("生成されたハッシュはCheckPasswordで検証できる", func(t *testing.T) {
		t.Parallel()

		password := "mySecurePassword456"
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword()のエラー = %v", err)
		}

		err = CheckPassword(hash, password)
		if err != nil {
			t.Errorf("正しいパスワードでのCheckPassword()のエラー = %v、期待値 = nil", err)
		}
	})

	t.Run("同じパスワードでも毎回異なるハッシュが生成される", func(t *testing.T) {
		t.Parallel()

		password := "samePassword789"
		hash1, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword()のエラー = %v", err)
		}

		hash2, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword()のエラー = %v", err)
		}

		if hash1 == hash2 {
			t.Error("bcryptのソルトによりハッシュが異なることを期待したが、同じだった")
		}

		// どちらのハッシュも検証可能であること
		if err := CheckPassword(hash1, password); err != nil {
			t.Errorf("hash1でのCheckPassword()のエラー = %v、期待値 = nil", err)
		}
		if err := CheckPassword(hash2, password); err != nil {
			t.Errorf("hash2でのCheckPassword()のエラー = %v、期待値 = nil", err)
		}
	})

	t.Run("日本語を含むパスワードをハッシュ化できる", func(t *testing.T) {
		t.Parallel()

		password := "パスワード123"
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword()のエラー = %v", err)
		}

		err = CheckPassword(hash, password)
		if err != nil {
			t.Errorf("CheckPassword()のエラー = %v、期待値 = nil", err)
		}
	})
}
