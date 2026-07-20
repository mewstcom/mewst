package model

import "time"

// Link is the domain model for a link card generated from a URL. It holds the
// canonical URL (unique), the source domain, and the OGP-derived title and
// image so a post can render a preview card without re-fetching the page.
//
// [Ja] Link は URL から生成されるリンクカードのドメインモデル。canonical URL
// (一意)・取得元ドメイン・OGP 由来のタイトルと画像を保持し、投稿がページを
// 再取得せずにプレビューカードを描画できるようにする。
type Link struct {
	ID           LinkID
	CanonicalURL string
	Domain       string
	Title        string
	ImageURL     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
