package usecase

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
)

// ExportObjectKeyPrefix is the object key prefix that scopes every export
// object in the shared bucket. Export code must list objects only under this
// prefix so that objects stored by other features are never enumerated or
// deleted.
//
// [Ja] ExportObjectKeyPrefix は共有バケット内のエクスポートオブジェクトを
// 限定するオブジェクトキーのプレフィックス。エクスポートのコードはこの
// プレフィックス配下だけを一覧し、他機能が保存したオブジェクトを列挙・削除
// しないようにする。
const ExportObjectKeyPrefix = "exports/"

// ExportObjectKey returns the deterministic object key for an export zip:
// exports/{profile_id}/{export_id}.zip. Retries of the same export overwrite
// the same key, so a retried generation never leaves partial objects under
// other keys.
//
// [Ja] ExportObjectKey はエクスポート zip の決定的なオブジェクトキー
// (exports/{profile_id}/{export_id}.zip) を返す。同じエクスポートのリトライは
// 同じキーを上書きするため、リトライされた生成が別のキーへ部分的な
// オブジェクトを残すことはない。
func ExportObjectKey(profileID model.ProfileID, exportID model.ExportID) string {
	return ExportObjectKeyPrefix + profileID.String() + "/" + exportID.String() + ".zip"
}

// ParseExportObjectKey validates that key follows the export object key
// convention and returns the embedded profile and export IDs. Every other
// key is rejected with an error, so callers that delete or reconcile objects
// never operate outside exports/{profile_id}/{export_id}.zip.
//
// [Ja] ParseExportObjectKey は key がエクスポートのオブジェクトキー規約に
// 従うことを検証し、埋め込まれたプロフィール ID とエクスポート ID を返す。
// 規約外のキーはすべてエラーとして拒否するため、オブジェクトを削除・照合する
// 呼び出し側が exports/{profile_id}/{export_id}.zip の外を操作することはない。
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
		return model.ProfileID{}, model.ExportID{}, fmt.Errorf("オブジェクトキーのプロフィール ID が不正 (key: %s): %w", key, err)
	}

	exportUUID, err := parseCanonicalUUID(exportPart)
	if err != nil {
		return model.ProfileID{}, model.ExportID{}, fmt.Errorf("オブジェクトキーのエクスポート ID が不正 (key: %s): %w", key, err)
	}

	return model.ProfileID(profileUUID), model.ExportID(exportUUID), nil
}

// parseCanonicalUUID parses s strictly as the canonical lowercase hyphenated
// UUID form. uuid.Parse alone also accepts braced, URN and unhyphenated
// encodings, which would give a single object several valid key spellings.
//
// [Ja] parseCanonicalUUID は s を正規の小文字ハイフン区切り UUID 形式として
// 厳密にパースする。uuid.Parse 単体は波括弧・URN・ハイフンなしの表現も
// 受け付けてしまい、1 つのオブジェクトに複数の有効なキー表記が生まれてしまう。
func parseCanonicalUUID(s string) (uuid.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, err
	}
	if u.String() != s {
		return uuid.UUID{}, fmt.Errorf("正規形の UUID ではない: %s", s)
	}
	return u, nil
}
