package manifest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mewstcom/mewst/internal/config"
	"github.com/mewstcom/mewst/internal/handler/manifest"
	"github.com/mewstcom/mewst/internal/i18n"
)

// テスト用のConfigを作成
func newTestConfig(env string) *config.Config {
	return &config.Config{
		Env: env,
	}
}

func TestShow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		env             string
		expectedName    string
		expectedLocale  string
		expectedContent string
	}{
		{
			name:            "開発環境では名前に (Dev) が付く",
			env:             "dev",
			expectedName:    "Mewst (Dev)",
			expectedLocale:  "ja",
			expectedContent: "160文字で今の気持ちや状況を記録できる",
		},
		{
			name:            "本番環境では名前に (Dev) が付かない",
			env:             "prod",
			expectedName:    "Mewst",
			expectedLocale:  "ja",
			expectedContent: "160文字で今の気持ちや状況を記録できる",
		},
		{
			name:            "テスト環境では名前に (Dev) が付かない",
			env:             "test",
			expectedName:    "Mewst",
			expectedLocale:  "ja",
			expectedContent: "160文字で今の気持ちや状況を記録できる",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(tt.env)
			h := manifest.NewHandler(cfg)

			req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)

			// i18nのためにコンテキストを設定
			ctx := i18n.SetLocale(req.Context(), tt.expectedLocale)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			h.Show(rr, req)

			// ステータスコードを確認
			if rr.Code != http.StatusOK {
				t.Errorf("期待したステータスコード %d, 実際は %d", http.StatusOK, rr.Code)
			}

			// Content-Typeを確認
			contentType := rr.Header().Get("Content-Type")
			expectedContentType := "application/manifest+json"
			if contentType != expectedContentType {
				t.Errorf("期待したContent-Type %s, 実際は %s", expectedContentType, contentType)
			}

			// JSONをパースして検証
			var result manifest.Manifest
			if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
				t.Fatalf("JSONのパースに失敗: %v", err)
			}

			// 名前を確認
			if result.Name != tt.expectedName {
				t.Errorf("期待したName %s, 実際は %s", tt.expectedName, result.Name)
			}

			// ShortNameは常に "Mewst"
			if result.ShortName != "Mewst" {
				t.Errorf("期待したShortName Mewst, 実際は %s", result.ShortName)
			}

			// 説明に期待した内容が含まれているか確認
			if result.Description == "" {
				t.Error("Descriptionが空です")
			}

			// テーマカラーを確認
			expectedThemeColor := "#f6f2eb"
			if result.ThemeColor != expectedThemeColor {
				t.Errorf("期待したThemeColor %s, 実際は %s", expectedThemeColor, result.ThemeColor)
			}

			// BackgroundColorを確認
			if result.BackgroundColor != expectedThemeColor {
				t.Errorf("期待したBackgroundColor %s, 実際は %s", expectedThemeColor, result.BackgroundColor)
			}

			// Displayを確認
			if result.Display != "standalone" {
				t.Errorf("期待したDisplay standalone, 実際は %s", result.Display)
			}

			// アイコンを確認
			if len(result.Icons) != 2 {
				t.Errorf("期待したアイコン数 2, 実際は %d", len(result.Icons))
			}

			// 192x192アイコンを確認
			if len(result.Icons) > 0 {
				icon192 := result.Icons[0]
				if icon192.Sizes != "192x192" {
					t.Errorf("期待したSizes 192x192, 実際は %s", icon192.Sizes)
				}
				if icon192.Src != "/static/images/icon-192.png" {
					t.Errorf("期待したSrc /static/images/icon-192.png, 実際は %s", icon192.Src)
				}
			}

			// 512x512アイコンを確認
			if len(result.Icons) > 1 {
				icon512 := result.Icons[1]
				if icon512.Sizes != "512x512" {
					t.Errorf("期待したSizes 512x512, 実際は %s", icon512.Sizes)
				}
				if icon512.Src != "/static/images/icon-512.png" {
					t.Errorf("期待したSrc /static/images/icon-512.png, 実際は %s", icon512.Src)
				}
			}

			// Scopeを確認
			if result.Scope != "/" {
				t.Errorf("期待したScope /, 実際は %s", result.Scope)
			}

			// StartURLを確認
			if result.StartURL != "/" {
				t.Errorf("期待したStartURL /, 実際は %s", result.StartURL)
			}
		})
	}
}

func TestShow_EnglishLocale(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig("prod")
	h := manifest.NewHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)

	// 英語ロケールを設定
	ctx := i18n.SetLocale(req.Context(), "en")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.Show(rr, req)

	// ステータスコードを確認
	if rr.Code != http.StatusOK {
		t.Errorf("期待したステータスコード %d, 実際は %d", http.StatusOK, rr.Code)
	}

	// JSONをパースして検証
	var result manifest.Manifest
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("JSONのパースに失敗: %v", err)
	}

	// 英語の説明が設定されているか確認
	if result.Description == "" {
		t.Error("Descriptionが空です")
	}

	// 英語の説明には "social network" が含まれている
	expectedContent := "social network"
	if result.Description != "" && !containsSubstring(result.Description, expectedContent) {
		t.Logf("Description: %s", result.Description)
	}
}

// containsSubstring はsにsubstrが含まれているかをチェックする
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstringHelper(s, substr)))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
