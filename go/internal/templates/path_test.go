package templates_test

import (
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates"
)

func TestStaticPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  templates.Path
		want templates.Path
	}{
		{name: "RootPath", got: templates.RootPath(), want: "/"},
		{name: "HomePath", got: templates.HomePath(), want: "/home"},
		{name: "SearchPath", got: templates.SearchPath(), want: "/search"},
		{name: "NewPostPath", got: templates.NewPostPath(), want: "/new"},
		{name: "NotificationListPath", got: templates.NotificationListPath(), want: "/notifications"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestProfilePath(t *testing.T) {
	t.Parallel()

	if got := templates.ProfilePath("alice"); got != "/@alice" {
		t.Errorf("ProfilePath(%q) = %q, want %q", "alice", got, "/@alice")
	}
}
