package viewmodel

import (
	"github.com/mewstcom/mewst/go/internal/model"
)

// NavbarItem identifies a menu item in the authenticated navbar.
//
// [Ja] 認証後 navbar のメニュー項目を識別する型。
type NavbarItem string

const (
	// NavbarItemNone marks a page that has no navbar item of its own (e.g.
	// /settings). It is the zero value, so the navbar renders with nothing
	// active; naming it lets call sites state that intent instead of passing "".
	//
	// [Ja] NavbarItemNone は navbar の項目を持たないページ (例: /settings) を表す。
	// ゼロ値であり navbar はどの項目もアクティブにせず描画する。定数として名前を
	// 与えることで、呼び出し側が "" を渡す代わりにその意図を明示できる。
	NavbarItemNone NavbarItem = ""
	// NavbarItemHome is the home menu item (/home).
	// [Ja] home メニュー項目 (/home)
	NavbarItemHome NavbarItem = "home"
	// NavbarItemSearch is the search menu item (/search).
	// [Ja] search メニュー項目 (/search)
	NavbarItemSearch NavbarItem = "search"
	// NavbarItemNew is the new post menu item (/new).
	// [Ja] 新規投稿メニュー項目 (/new)
	NavbarItemNew NavbarItem = "new"
	// NavbarItemNotification is the notification menu item (/notifications).
	// [Ja] 通知メニュー項目 (/notifications)
	NavbarItemNotification NavbarItem = "notification"
	// NavbarItemProfile is the profile menu item (/@{atname}).
	// [Ja] プロフィールメニュー項目 (/@{atname})
	NavbarItemProfile NavbarItem = "profile"
)

// Navbar holds the data needed to render the authenticated navbar.
//
// Atname is the current user's atname, used to build the profile link
// (/@{atname}). ActiveItem indicates which menu item should be shown as
// active (filled icon) on the current page.
//
// [Ja] 認証後 navbar の描画に必要なデータを保持する。
//
// Atname は現在ユーザーの atname で、プロフィールリンク (/@{atname}) の
// 生成に使う。ActiveItem は現在のページでどのメニュー項目をアクティブ
// (塗りつぶしアイコン) として表示するかを示す。
type Navbar struct {
	Atname     string
	ActiveItem NavbarItem
}

// NewNavbar builds a Navbar viewmodel from the current profile and the
// menu item that is active on the current page. A nil profile yields an
// empty atname so callers never need to special-case unauthenticated state.
//
// [Ja] 現在のプロフィールと、現在のページでアクティブなメニュー項目から
// Navbar viewmodel を生成する。profile が nil の場合は atname を空にする
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

// IsActive reports whether the given menu item is the active one.
//
// [Ja] 指定したメニュー項目が現在アクティブかどうかを返す。
func (n Navbar) IsActive(item NavbarItem) bool {
	return n.ActiveItem == item
}
