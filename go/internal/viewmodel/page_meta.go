// Package viewmodel はプレゼンテーション層のデータ変換を担当するパッケージ
package viewmodel

// PageMeta はページのメタ情報を保持する構造体
type PageMeta struct {
	Title        string // ページタイトル（<title>タグ、og:title用）
	Description  string // ページ説明（descriptionメタタグ、og:description用）
	AssetVersion string // アセットのバージョン（キャッシュバスティング用）
	OGType       string // og:typeの値（"website", "article"など）
	OGURL        string // og:urlの値（canonicalと同じ）
	OGImage      string // og:imageの値
	OGLocale     string // og:localeの値（"ja_JP", "en_US"など）
}
