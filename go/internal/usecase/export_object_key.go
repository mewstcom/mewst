package usecase

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
)

// ExportObjectKeyPrefixは共有バケット内のエクスポートオブジェクトを
// 限定するオブジェクトキーのプレフィックス。エクスポートのコードはこの
// プレフィックス配下だけを一覧し、他機能が保存したオブジェクトを列挙・削除
// しないようにする。
const ExportObjectKeyPrefix = "exports/"

// ExportObjectKeyはエクスポートzipの決定的なオブジェクトキー
// (exports/{profile_id}/{export_id}.zip) を返す。同じエクスポートのリトライは
// 同じキーを上書きするため、リトライされた生成が別のキーへ部分的な
// オブジェクトを残すことはない。
func ExportObjectKey(profileID model.ProfileID, exportID model.ExportID) string {
	return ExportObjectKeyPrefix + profileID.String() + "/" + exportID.String() + ".zip"
}

// ParseExportObjectKeyはkeyがエクスポートのオブジェクトキー規約に
// 従うことを検証し、埋め込まれたプロフィールIDとエクスポートIDを返す。
// 規約外のキーはすべてエラーとして拒否するため、オブジェクトを削除・照合する
// 呼び出し側がexports/{profile_id}/{export_id}.zipの外を操作することはない。
func ParseExportObjectKey(key string) (model.ProfileID, model.ExportID, error) {
	rest, ok := strings.CutPrefix(key, ExportObjectKeyPrefix)
	if !ok {
		return model.ProfileID{}, model.ExportID{}, fmt.Errorf("エクスポートのオブジェクトキーではない (key: %s)", key)
	}

	profilePart, filePart, ok := strings.Cut(rest, "/")
	if !ok {
		return model.ProfileID{}, model.ExportID{}, fmt.Errorf("オブジェクトキーの形式が不正 (key: %s)", key)
	}

	exportPart, ok := strings.CutSuffix(filePart, ".zip")
	if !ok {
		return model.ProfileID{}, model.ExportID{}, fmt.Errorf("オブジェクトキーの形式が不正 (key: %s)", key)
	}

	profileUUID, err := parseCanonicalUUID(profilePart)
	if err != nil {
		return model.ProfileID{}, model.ExportID{}, fmt.Errorf("オブジェクトキーのプロフィールIDが不正 (key: %s): %w", key, err)
	}

	exportUUID, err := parseCanonicalUUID(exportPart)
	if err != nil {
		return model.ProfileID{}, model.ExportID{}, fmt.Errorf("オブジェクトキーのエクスポートIDが不正 (key: %s): %w", key, err)
	}

	return model.ProfileID(profileUUID), model.ExportID(exportUUID), nil
}

// parseCanonicalUUIDはsを正規の小文字ハイフン区切りUUID形式として
// 厳密にパースする。uuid.Parse単体は波括弧・URN・ハイフンなしの表現も
// 受け付けてしまい、1つのオブジェクトに複数の有効なキー表記が生まれてしまう。
func parseCanonicalUUID(s string) (uuid.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, err
	}
	if u.String() != s {
		return uuid.UUID{}, fmt.Errorf("正規形のUUIDではない: %s", s)
	}
	return u, nil
}
