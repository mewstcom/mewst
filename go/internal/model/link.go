package model

import "time"

// LinkはURLから生成されるリンクカードのドメインモデル。canonical URL
// (一意)・取得元ドメイン・OGP由来のタイトルと画像を保持し、投稿がページを
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
