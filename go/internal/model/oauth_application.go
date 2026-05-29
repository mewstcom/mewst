package model

import "time"

// MewstWebUID is the uid of the mewst-web OAuth application, used to attribute
// posts created from the web frontend. It mirrors the Rails
// OauthApplication::MEWST_WEB_UID constant and is resolved via
// OauthApplicationRepository.FindByUID.
//
// [Ja] MewstWebUID は web フロントエンドから作成された投稿に紐づける mewst-web
// OAuth アプリケーションの uid。Rails の OauthApplication::MEWST_WEB_UID 定数に
// 対応し、OauthApplicationRepository.FindByUID で解決する。
const MewstWebUID = "mewst-web"

// OauthApplication is the domain model for an OAuth application.
// [Ja] OauthApplication は OAuth アプリケーションのドメインモデル。
type OauthApplication struct {
	ID           OauthApplicationID
	Name         string
	UID          string
	Secret       string
	RedirectURI  string
	Scopes       string
	Confidential bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
