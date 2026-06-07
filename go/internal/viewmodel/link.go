package viewmodel

import (
	"net/url"

	"github.com/mewstcom/mewst/go/internal/model"
)

// shortenHostAndPathLength is the maximum rune length of the host-and-path
// label shown on the add-link-card button, mirroring the Rails
// Url#shorten_host_and_path default (truncate to 25 characters).
//
// [Ja] shortenHostAndPathLength はリンクカード追加ボタンに表示する host + path
// ラベルの最大 rune 数。Rails の Url#shorten_host_and_path のデフォルト
// (25 文字に truncate) に対応する。
const shortenHostAndPathLength = 25

// Link is the view model for rendering a link card.
// [Ja] Link はリンクカードを描画するための view model。
type Link struct {
	CanonicalURL string
	Domain       string
	Title        string
	ImageURL     string
}

// NewLink builds a Link view model from the domain model.
// [Ja] NewLink はドメインモデルから Link view model を生成する。
func NewLink(link *model.Link) Link {
	return Link{
		CanonicalURL: link.CanonicalURL,
		Domain:       link.Domain,
		Title:        link.Title,
		ImageURL:     link.ImageURL,
	}
}

// ShortenHostAndPath returns the host and path of the URL truncated to 25
// characters for display on the add-link-card button, mirroring the Rails
// Url#shorten_host_and_path (host without port + path, String#truncate with a
// "..." omission). An unparsable URL or one without a host yields "".
//
// [Ja] ShortenHostAndPath は URL の host + path を 25 文字に切り詰めて返す
// (リンクカード追加ボタンの表示用)。Rails の Url#shorten_host_and_path
// (ポートを除く host + path を String#truncate で "..." 付きに省略) に対応する。
// パース不能な URL や host を持たない URL は "" を返す。
func ShortenHostAndPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return ""
	}

	hostAndPath := []rune(u.Hostname() + u.Path)
	if len(hostAndPath) <= shortenHostAndPathLength {
		return string(hostAndPath)
	}
	return string(hostAndPath[:shortenHostAndPathLength-len("...")]) + "..."
}
