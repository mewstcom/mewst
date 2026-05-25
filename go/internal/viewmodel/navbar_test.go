package viewmodel

import (
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
)

func TestNewNavbar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		profile    *model.Profile
		activeItem NavbarItem
		wantAtname string
	}{
		{
			name:       "プロフィールから atname とアクティブ項目を設定する",
			profile:    &model.Profile{Atname: "alice"},
			activeItem: NavbarItemNew,
			wantAtname: "alice",
		},
		{
			name:       "プロフィールが nil の場合は atname を空にする",
			profile:    nil,
			activeItem: NavbarItemHome,
			wantAtname: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			navbar := NewNavbar(tt.profile, tt.activeItem)

			if navbar.Atname != tt.wantAtname {
				t.Errorf("NewNavbar().Atname = %q, want %q", navbar.Atname, tt.wantAtname)
			}
			if navbar.ActiveItem != tt.activeItem {
				t.Errorf("NewNavbar().ActiveItem = %q, want %q", navbar.ActiveItem, tt.activeItem)
			}
		})
	}
}

func TestNavbar_IsActive(t *testing.T) {
	t.Parallel()

	navbar := NewNavbar(&model.Profile{Atname: "bob"}, NavbarItemNew)

	if !navbar.IsActive(NavbarItemNew) {
		t.Error("IsActive(NavbarItemNew) = false, want true")
	}
	if navbar.IsActive(NavbarItemHome) {
		t.Error("IsActive(NavbarItemHome) = true, want false")
	}
}
