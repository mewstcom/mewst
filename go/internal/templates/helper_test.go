package templates_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

func TestDeref(t *testing.T) {
	t.Parallel()

	t.Run("intのポインタが非nilの場合は値を返す", func(t *testing.T) {
		t.Parallel()

		v := 42
		got := templates.Deref(&v)
		if got != 42 {
			t.Errorf("Deref(&42) = %d, want 42", got)
		}
	})

	t.Run("intのポインタがnilの場合はゼロ値を返す", func(t *testing.T) {
		t.Parallel()

		var p *int
		got := templates.Deref(p)
		if got != 0 {
			t.Errorf("Deref(nil) = %d, want 0", got)
		}
	})

	t.Run("stringのポインタが非nilの場合は値を返す", func(t *testing.T) {
		t.Parallel()

		s := "hello"
		got := templates.Deref(&s)
		if got != "hello" {
			t.Errorf("Deref(&\"hello\") = %q, want \"hello\"", got)
		}
	})

	t.Run("stringのポインタがnilの場合は空文字列を返す", func(t *testing.T) {
		t.Parallel()

		var p *string
		got := templates.Deref(p)
		if got != "" {
			t.Errorf("Deref(nil) = %q, want \"\"", got)
		}
	})

	t.Run("構造体のポインタが非nilの場合は値を返す", func(t *testing.T) {
		t.Parallel()

		type point struct {
			X, Y int
		}

		v := point{X: 1, Y: 2}
		got := templates.Deref(&v)
		if got.X != 1 || got.Y != 2 {
			t.Errorf("Deref(&point{1,2}) = %+v, want {X:1 Y:2}", got)
		}
	})

	t.Run("構造体のポインタがnilの場合はゼロ値を返す", func(t *testing.T) {
		t.Parallel()

		type point struct {
			X, Y int
		}

		var p *point
		got := templates.Deref(p)
		if got.X != 0 || got.Y != 0 {
			t.Errorf("Deref(nil) = %+v, want {X:0 Y:0}", got)
		}
	})
}

func TestIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		iconName       viewmodel.IconName
		class          []string
		wantContains   []string
		wantNoContains []string
	}{
		{
			name:         "クラス指定なしでcustomアイコン(logo)が描画される",
			iconName:     "logo",
			class:        nil,
			wantContains: []string{"<svg", "viewBox=\"0 0 512 512\""},
			// クラス未指定時はclass属性が挿入されない
			wantNoContains: []string{`class="`},
		},
		{
			name:         "クラス指定ありでphosphorアイコン(arrow-right-regular)が描画される",
			iconName:     "arrow-right-regular",
			class:        []string{"icon-sm"},
			wantContains: []string{"<svg", `class="icon-sm"`, "viewBox=\"0 0 256 256\""},
		},
		{
			name:         "未定義のアイコン名はinfo-regularにフォールバックする",
			iconName:     "nonexistent-icon-xyz",
			class:        nil,
			wantContains: []string{"<svg", "viewBox=\"0 0 256 256\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			component := templates.Icon(tt.iconName, tt.class...)

			var buf bytes.Buffer
			if err := component.Render(context.Background(), &buf); err != nil {
				t.Fatalf("Icon.Render() error = %v", err)
			}

			html := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("Icon(%q) = %q, want to contain %q", tt.iconName, html, want)
				}
			}
			for _, notWant := range tt.wantNoContains {
				if strings.Contains(html, notWant) {
					t.Errorf("Icon(%q) = %q, want NOT to contain %q", tt.iconName, html, notWant)
				}
			}
		})
	}
}

func TestIcon_未定義のアイコン名はinfo_regularの内容と一致する(t *testing.T) {
	t.Parallel()

	var fallbackBuf, undefinedBuf bytes.Buffer
	if err := templates.Icon("info-regular").Render(context.Background(), &fallbackBuf); err != nil {
		t.Fatalf("Icon(\"info-regular\") error = %v", err)
	}
	if err := templates.Icon("does-not-exist").Render(context.Background(), &undefinedBuf); err != nil {
		t.Fatalf("Icon(\"does-not-exist\") error = %v", err)
	}

	if fallbackBuf.String() != undefinedBuf.String() {
		t.Errorf("undefined icon should fall back to info-regular, got = %q, want = %q", undefinedBuf.String(), fallbackBuf.String())
	}
}
