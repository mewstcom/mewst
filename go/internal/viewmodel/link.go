package viewmodel

import (
	"net/url"

	"github.com/mewstcom/mewst/go/internal/model"
)

// shortenHostAndPathLengthはリンクカード追加ボタンに表示するhost + path
// ラベルの最大rune数。RailsのUrl#shorten_host_and_pathのデフォルト
// (25文字にtruncate) に対応する。
const shortenHostAndPathLength = 25

// Linkはリンクカードを描画するためのview model。
type Link struct {
	CanonicalURL string
	Domain       string
	Title        string
	ImageURL     string
}

// NewLinkはドメインモデルからLink view modelを生成する。
func NewLink(link *model.Link) Link {
	return Link{
		CanonicalURL: link.CanonicalURL,
		Domain:       link.Domain,
		Title:        link.Title,
		ImageURL:     link.ImageURL,
	}
}

// ShortenHostAndPathはURLのhost + pathを25文字に切り詰めて返す
// (リンクカード追加ボタンの表示用)。RailsのUrl#shorten_host_and_path
// (ポートを除くhost + pathをString#truncateで "..." 付きに省略) に対応する。
// パース不能なURLやhostを持たないURLは "" を返す。
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
