package viewmodel

import (
	"github.com/mewstcom/mewst/go/internal/model"
)

// NavbarItemは認証後navbarのメニュー項目を識別する型。
type NavbarItem string

const (
	// NavbarItemNoneはnavbarの項目を持たないページ (例: /settings) を表す。
	// ゼロ値でありnavbarはどの項目もアクティブにせず描画する。定数として名前を
	// 与えることで、呼び出し側が "" を渡す代わりにその意図を明示できる。
	NavbarItemNone NavbarItem = ""
	// NavbarItemHomeはhomeメニュー項目 (/home)
	NavbarItemHome NavbarItem = "home"
	// NavbarItemSearchはsearchメニュー項目 (/search)
	NavbarItemSearch NavbarItem = "search"
	// NavbarItemNewは新規投稿メニュー項目 (/new)
	NavbarItemNew NavbarItem = "new"
	// NavbarItemNotificationは通知メニュー項目 (/notifications)
	NavbarItemNotification NavbarItem = "notification"
	// NavbarItemProfileはプロフィールメニュー項目 (/@{atname})
	NavbarItemProfile NavbarItem = "profile"
)

// Navbarは認証後navbarの描画に必要なデータを保持する。
//
// Atnameは現在ユーザーのatnameで、プロフィールリンク (/@{atname}) の
// 生成に使う。ActiveItemは現在のページでどのメニュー項目をアクティブ
// (塗りつぶしアイコン) として表示するかを示す。
type Navbar struct {
	Atname     string
	ActiveItem NavbarItem
}

// NewNavbarは現在のプロフィールと、現在のページでアクティブなメニュー項目から
// Navbar viewmodelを生成する。profileがnilの場合はatnameを空にする
// ため、呼び出し元で未認証状態を特別扱いする必要はない。
func NewNavbar(profile *model.Profile, activeItem NavbarItem) Navbar {
	atname := ""
	if profile != nil {
		atname = profile.Atname
	}
	return Navbar{
		Atname:     atname,
		ActiveItem: activeItem,
	}
}

// IsActiveは指定したメニュー項目が現在アクティブかどうかを返す。
func (n Navbar) IsActive(item NavbarItem) bool {
	return n.ActiveItem == item
}
