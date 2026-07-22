package templates

// Path represents a URL path within the app. Centralizing path construction in
// one place keeps route strings out of templates and gives a single point to
// update when a route changes (mirroring the Wikino path helpers).
//
// [Ja] Path はアプリ内の URL パスを表す型。パス生成を一箇所に集約することで、
// ルート文字列をテンプレートから排除し、ルート変更時の修正点を一本化する
// (Wikino のパスヘルパーに倣ったもの)。
type Path string

// RootPath returns the path to the site root (the navbar logo links here).
// [Ja] RootPath はサイトのルートへのパスを返す (navbar のロゴのリンク先)。
func RootPath() Path {
	return Path("/")
}

// HomePath returns the path to the home timeline.
// [Ja] HomePath はホームタイムラインへのパスを返す。
func HomePath() Path {
	return Path("/home")
}

// SearchPath returns the path to the search page.
// [Ja] SearchPath は検索ページへのパスを返す。
func SearchPath() Path {
	return Path("/search")
}

// NewPostPath returns the path to the new post form (GET /new).
// [Ja] NewPostPath は新規投稿フォーム (GET /new) へのパスを返す。
func NewPostPath() Path {
	return Path("/new")
}

// NotificationListPath returns the path to the notification list.
// [Ja] NotificationListPath は通知一覧へのパスを返す。
func NotificationListPath() Path {
	return Path("/notifications")
}

// ProfilePath returns the path to the given user's profile page (/@{atname}).
// [Ja] ProfilePath は指定ユーザーのプロフィールページ (/@{atname}) へのパスを返す。
func ProfilePath(atname string) Path {
	return Path("/@" + atname)
}

// SettingListPath returns the path to the settings menu.
//
// [Ja] SettingListPath は設定メニューへのパスを返す。
func SettingListPath() Path {
	return Path("/settings")
}

// SettingProfilePath returns the path to the profile settings page.
//
// [Ja] SettingProfilePath はプロフィール設定ページへのパスを返す。
func SettingProfilePath() Path {
	return Path("/settings/profile")
}

// SettingUserPath returns the path to the user settings page.
//
// [Ja] SettingUserPath はユーザー設定ページへのパスを返す。
func SettingUserPath() Path {
	return Path("/settings/user")
}

// SettingEmailPath returns the path to the email settings page.
//
// [Ja] SettingEmailPath はメールアドレス設定ページへのパスを返す。
func SettingEmailPath() Path {
	return Path("/settings/email")
}

// SignOutPath returns the path that the sign-out form posts to.
//
// [Ja] SignOutPath はログアウトフォームの送信先パスを返す。
func SignOutPath() Path {
	return Path("/sign_out")
}
