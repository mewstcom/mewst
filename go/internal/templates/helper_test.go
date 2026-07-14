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
			wantContains: []string{"<svg", "viewBox=\"0 0 700 700\""},
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

	t.Run("未定義のアイコン名はinfo-regularと完全一致する", func(t *testing.T) {
		t.Parallel()

		// An undefined icon name must fall back to info-regular. Compared by exact
		// equality (not substring) so a future change that makes the fallback emit
		// anything other than info-regular's exact SVG is caught.
		//
		// [Ja] 未定義のアイコン名は info-regular にフォールバックしなければならない。
		// 部分一致ではなく完全一致で比較することで、将来フォールバックが
		// info-regular とは異なる SVG を返すようになった場合に検出できる。
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
	})
}

func TestIcon_MingcuteIcons(t *testing.T) {
	t.Parallel()

	// Each navbar icon ported from the Rails version must be registered and
	// rendered as its own SVG: not falling back to info-regular, and not mixed
	// up with another icon (e.g. its line/fill counterpart). The value is a
	// substring unique to that icon's visible `content` path, taken from the
	// Rails source SVG, so a copy-paste swap between two icons is detected.
	//
	// [Ja] Rails 版から移植した navbar 用アイコンが、それぞれ固有の SVG として
	// 登録・描画されることを確認する。info-regular へのフォールバックや、
	// line / fill など別アイコンとの取り違えに落ちていないこと。値は Rails の
	// 元 SVG から取った、そのアイコンの表示用 `content` path 固有の部分文字列で、
	// アイコン間のコピペ取り違えを検出できる。
	pathFragments := map[viewmodel.IconName]string{
		"home_4_line":       `d="M10.8 2.65`,
		"home_4_fill":       `d="M13.2 2.65`,
		"search_line":       `d="M10.5 2a8.5 8.5 0 1 0 5.262`,
		"search_fill":       `d="M10.5 2a8.5 8.5 0 0 1 6.676`,
		"edit_4_line":       `d="M5 2a2 2 0 0 0-2 2v15`,
		"edit_4_fill":       `d="m14.535 12.225`,
		"notification_line": `d="M5.00016,9 C5.00016`,
		"notification_fill": `d="M12.0002,2 C8.13417`,
		"user_4_line":       `d="M12 2c5.523`,
		"user_4_fill":       `d="M12 2C6.477`,
	}

	for name, pathFragment := range pathFragments {
		t.Run(string(name), func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := templates.Icon(name).Render(context.Background(), &buf); err != nil {
				t.Fatalf("Icon(%q).Render() error = %v", name, err)
			}

			html := buf.String()
			// `<svg`, the `content` class and the 24x24 viewBox confirm a ported
			// mingcute icon is rendered instead of the info-regular fallback
			// (256x256, no content class); the per-icon path fragment additionally
			// confirms the correct icon is registered under this key.
			//
			// [Ja] `<svg`・`content` クラス・24x24 の viewBox は、info-regular
			// フォールバック (256x256・content クラスなし) ではなく移植した
			// mingcute アイコンが描画されていることを示す。加えてアイコン固有の
			// path 片により、このキーに正しいアイコンが登録されていることを確認する。
			for _, want := range []string{"<svg", `class="content"`, `viewBox="0 0 24 24"`, pathFragment} {
				if !strings.Contains(html, want) {
					t.Errorf("Icon(%q) = %q, want to contain %q", name, html, want)
				}
			}
		})
	}
}
