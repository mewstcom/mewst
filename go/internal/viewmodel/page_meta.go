// Package viewmodel はプレゼンテーション層のデータ変換を担当するパッケージ
package viewmodel

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/i18n"
)

// サイト名のサフィックス
const siteSuffix = " | Mewst"

// PageMeta はページのメタ情報を保持する構造体
type PageMeta struct {
	Title        string // ページタイトル (<title>タグ、og:title用)
	Description  string // ページ説明 (descriptionメタタグ、og:description用)
	AssetVersion string // アセットのバージョン (キャッシュバスティング用)
	OGType       string // og:typeの値 ("website", "article"など)
	OGURL        string // og:urlの値 (canonicalと同じ)
	OGImage      string // og:imageの値
	OGLocale     string // og:localeの値 ("ja_JP", "en_US"など)
}

// DefaultPageMeta はデフォルトのメタ情報を返す
// コンテキストから検出された言語に応じて、タイトルと説明が自動的に切り替わる
// Titleには自動的に " | Mewst" サフィックスが付加される
func DefaultPageMeta(ctx context.Context, cfg *config.Config) PageMeta {
	ogImageURL := cfg.AppURL() + "/static/images/og-image.png"
	title := i18n.T(ctx, "default_title") + siteSuffix

	return PageMeta{
		Title:        title,
		Description:  i18n.T(ctx, "default_description"),
		AssetVersion: cfg.GetAssetVersion(),
		OGType:       "website",
		OGURL:        "",
		OGImage:      ogImageURL,
		OGLocale:     ogLocaleFromLocale(i18n.GetLocale(ctx)),
	}
}

// ogLocaleFromLocale はロケール文字列からOGP用のロケール文字列に変換する
func ogLocaleFromLocale(locale string) string {
	switch locale {
	case "ja":
		return "ja_JP"
	case "en":
		return "en_US"
	default:
		return "ja_JP"
	}
}

// SetTitle はタイトルを設定する (" | Mewst" サフィックス付き)
// 通常のページで使用する
func (p *PageMeta) SetTitle(ctx context.Context, titleKey string) {
	p.Title = i18n.T(ctx, titleKey) + siteSuffix
}

// SetTitleWithoutSuffix はタイトルを設定する (サフィックスなし)
// トップページなど、サフィックスが不要なページで使用する
func (p *PageMeta) SetTitleWithoutSuffix(ctx context.Context, titleKey string) {
	p.Title = i18n.T(ctx, titleKey)
}

// SetOGURL はOGURLを設定する
// pathにはクエリパラメータを除いたパスを指定する (canonical URLとして適切な形式)
func (p *PageMeta) SetOGURL(cfg *config.Config, path string) {
	p.OGURL = cfg.AppURL() + path
}
