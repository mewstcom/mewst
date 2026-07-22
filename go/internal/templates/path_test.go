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
		{name: "SettingListPath", got: templates.SettingListPath(), want: "/settings"},
		{name: "SettingProfilePath", got: templates.SettingProfilePath(), want: "/settings/profile"},
		{name: "SettingUserPath", got: templates.SettingUserPath(), want: "/settings/user"},
		{name: "SettingEmailPath", got: templates.SettingEmailPath(), want: "/settings/email"},
		{name: "SignOutPath", got: templates.SignOutPath(), want: "/sign_out"},
		{name: "CommunityPath", got: templates.CommunityPath(), want: "/community"},
		{name: "TermsPath", got: templates.TermsPath(), want: "/terms"},
		{name: "PrivacyPath", got: templates.PrivacyPath(), want: "/privacy"},
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
