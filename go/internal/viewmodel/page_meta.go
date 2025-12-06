// Package viewmodel はプレゼンテーション層のデータ変換を担当するパッケージ
package viewmodel

// PageMeta はページのメタ情報を保持する構造体
type PageMeta struct {
	Title        string // ページタイトル（<title>タグ用）
	Description  string // ページ説明（descriptionメタタグ用）
	AssetVersion string // アセットのバージョン（キャッシュバスティング用）
}
