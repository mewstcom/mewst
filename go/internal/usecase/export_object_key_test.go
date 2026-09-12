package usecase_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestExportObjectKey(t *testing.T) {
	t.Parallel()

	profileID := model.ProfileID(uuid.MustParse("0192a1b2-c3d4-7e5f-8a6b-7c8d9e0f1a2b"))
	exportID := model.ExportID(uuid.MustParse("0192f3e4-d5c6-7b8a-9f0e-1d2c3b4a5968"))

	got := usecase.ExportObjectKey(profileID, exportID)
	want := "exports/0192a1b2-c3d4-7e5f-8a6b-7c8d9e0f1a2b/0192f3e4-d5c6-7b8a-9f0e-1d2c3b4a5968.zip"
	if got != want {
		t.Errorf("ExportObjectKey() = %q, want %q", got, want)
	}
	if !strings.HasPrefix(got, usecase.ExportObjectKeyPrefix) {
		t.Errorf("ExportObjectKey() = %q, want prefix %q", got, usecase.ExportObjectKeyPrefix)
	}
}

func TestParseExportObjectKey_RoundTrip(t *testing.T) {
	t.Parallel()

	profileID := model.ProfileID(uuid.New())
	exportID := model.ExportID(uuid.New())

	gotProfileID, gotExportID, err := usecase.ParseExportObjectKey(usecase.ExportObjectKey(profileID, exportID))
	if err != nil {
		t.Fatalf("ParseExportObjectKey() error = %v", err)
	}
	if gotProfileID != profileID {
		t.Errorf("profile ID = %s, want %s", gotProfileID, profileID)
	}
	if gotExportID != exportID {
		t.Errorf("export ID = %s, want %s", gotExportID, exportID)
	}
}

func TestParseExportObjectKey_RejectsKeysOutsideConvention(t *testing.T) {
	t.Parallel()

	// Two canonical UUIDs used to build near-miss keys.
	//
	// [Ja] ニアミスなキーを組み立てるための正規形 UUID 2 つ。
	p := "0192a1b2-c3d4-7e5f-8a6b-7c8d9e0f1a2b"
	e := "0192f3e4-d5c6-7b8a-9f0e-1d2c3b4a5968"

	tests := []struct {
		name string
		key  string
	}{
		{name: "空文字列", key: ""},
		{name: "プレフィックスのみ", key: "exports/"},
		{name: "プレフィックスなし", key: p + "/" + e + ".zip"},
		{name: "他機能のプレフィックス", key: "images/" + p + "/" + e + ".zip"},
		{name: "エクスポート ID セグメントなし", key: "exports/" + p + ".zip"},
		{name: "余分なセグメント", key: "exports/" + p + "/" + e + "/" + e + ".zip"},
		{name: "拡張子なし", key: "exports/" + p + "/" + e},
		{name: "拡張子違い", key: "exports/" + p + "/" + e + ".txt"},
		{name: "二重拡張子", key: "exports/" + p + "/" + e + ".zip.zip"},
		{name: "エクスポート ID が空", key: "exports/" + p + "/.zip"},
		{name: "プロフィール ID が空", key: "exports//" + e + ".zip"},
		{name: "UUID ではないセグメント", key: "exports/profile/export.zip"},
		{name: "大文字の UUID", key: "exports/" + strings.ToUpper(p) + "/" + e + ".zip"},
		{name: "ハイフンなしの UUID 表現", key: "exports/" + strings.ReplaceAll(p, "-", "") + "/" + e + ".zip"},
		{name: "波括弧付きの UUID 表現", key: "exports/{" + p + "}/" + e + ".zip"},
		{name: "URN 形式の UUID 表現", key: "exports/urn:uuid:" + p + "/" + e + ".zip"},
		{name: "パストラバーサル", key: "exports/../" + e + ".zip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := usecase.ParseExportObjectKey(tt.key); err == nil {
				t.Errorf("ParseExportObjectKey(%q) error = nil, want error", tt.key)
			}
		})
	}
}
