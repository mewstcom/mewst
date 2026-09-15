package model

import "time"

// MewstWebUIDはwebフロントエンドから作成された投稿に紐づけるmewst-web
// OAuthアプリケーションのuid。RailsのOauthApplication::MEWST_WEB_UID定数に
// 対応し、OauthApplicationRepository.FindByUIDで解決する。
const MewstWebUID = "mewst-web"

// OauthApplicationはOAuthアプリケーションのドメインモデル。
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
