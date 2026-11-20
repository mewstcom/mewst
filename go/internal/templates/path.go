package templates

// Pathはアプリ内のURLパスを表す型。パス生成を一箇所に集約することで、
// ルート文字列をテンプレートから排除し、ルート変更時の修正点を一本化する
// (Wikinoのパスヘルパーに倣ったもの)。
type Path string

// RootPathはサイトのルートへのパスを返す (navbarのロゴのリンク先)。
func RootPath() Path {
	return Path("/")
}

// HomePathはホームタイムラインへのパスを返す。
func HomePath() Path {
	return Path("/home")
}

// SearchPathは検索ページへのパスを返す。
func SearchPath() Path {
	return Path("/search")
}

// NewPostPathは新規投稿フォーム (GET /new) へのパスを返す。
func NewPostPath() Path {
	return Path("/new")
}

// NotificationListPathは通知一覧へのパスを返す。
func NotificationListPath() Path {
	return Path("/notifications")
}

// ProfilePathは指定ユーザーのプロフィールページ (/@{atname}) へのパスを返す。
func ProfilePath(atname string) Path {
	return Path("/@" + atname)
}

// SettingListPathは設定メニューへのパスを返す。
func SettingListPath() Path {
	return Path("/settings")
}

// SettingProfilePathはプロフィール設定ページへのパスを返す。
func SettingProfilePath() Path {
	return Path("/settings/profile")
}

// SettingUserPathはユーザー設定ページへのパスを返す。
func SettingUserPath() Path {
	return Path("/settings/user")
}

// SettingEmailPathはメールアドレス設定ページへのパスを返す。
func SettingEmailPath() Path {
	return Path("/settings/email")
}

// SettingExportPathはエクスポートページへのパスを返す。完了メールが読み手を
// 送る先でもある。
func SettingExportPath() Path {
	return Path("/settings/export")
}

// SettingExportDownloadPathはプロフィールの最新の成功したエクスポートを
// ダウンロードするために予約したパスを返す。
func SettingExportDownloadPath() Path {
	return Path("/settings/export/download")
}

// SignOutPathはログアウトフォームの送信先パスを返す。
func SignOutPath() Path {
	return Path("/sign_out")
}

// CommunityPathはコミュニティページへのパスを返す。ページ自体はRails版が
// 応答するが、リンク元のフッターはGo版が描画する。
func CommunityPath() Path {
	return Path("/community")
}

// TermsPathは利用規約ページへのパスを返す。ページ自体はRails版が応答するが、
// リンク元のフッターはGo版が描画する。
func TermsPath() Path {
	return Path("/terms")
}

// PrivacyPathはプライバシーポリシーページへのパスを返す。ページ自体はRails版が
// 応答するが、リンク元のフッターはGo版が描画する。
func PrivacyPath() Path {
	return Path("/privacy")
}
