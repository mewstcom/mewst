package repository_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestLinkRepository_FindByCanonicalURL(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	canonicalURL := "https://example.com/find-by-canonical-url"
	linkID := testutil.NewLinkBuilder(t, tx).
		WithCanonicalURL(canonicalURL).
		WithDomain("example.com").
		WithTitle("Example Title").
		WithImageURL("https://example.com/og.png").
		Build()

	repo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))

	t.Run("canonical URLでリンクを取得できる", func(t *testing.T) {
		link, err := repo.FindByCanonicalURL(ctx, canonicalURL)
		if err != nil {
			t.Fatalf("FindByCanonicalURL()のエラー = %v", err)
		}
		if link == nil {
			t.Fatal("FindByCanonicalURL() = nil、リンクを期待")
		}
		if link.ID != linkID {
			t.Errorf("link.ID = %v、期待値 = %v", link.ID, linkID)
		}
		if link.CanonicalURL != canonicalURL {
			t.Errorf("link.CanonicalURL = %v、期待値 = %v", link.CanonicalURL, canonicalURL)
		}
		if link.Domain != "example.com" {
			t.Errorf("link.Domain = %v、期待値 = example.com", link.Domain)
		}
		if link.Title != "Example Title" {
			t.Errorf("link.Title = %v、期待値 = Example Title", link.Title)
		}
		if link.ImageURL != "https://example.com/og.png" {
			t.Errorf("link.ImageURL = %v、期待値 = https://example.com/og.png", link.ImageURL)
		}
	})

	t.Run("存在しないcanonical URLはnilを返す", func(t *testing.T) {
		link, err := repo.FindByCanonicalURL(ctx, "https://example.com/nonexistent")
		if err != nil {
			t.Errorf("FindByCanonicalURL()のエラー = %v、期待値 = nil", err)
		}
		if link != nil {
			t.Errorf("FindByCanonicalURL()のlink = %v、期待値 = nil", link)
		}
	})
}

func TestLinkRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	repo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))

	t.Run("リンクを作成できる", func(t *testing.T) {
		link, err := repo.Create(ctx, repository.CreateLinkInput{
			CanonicalURL: "https://example.com/created",
			Domain:       "example.com",
			Title:        "Created Link",
			ImageURL:     "https://example.com/created.png",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}

		if link.CanonicalURL != "https://example.com/created" {
			t.Errorf("link.CanonicalURL = %v、期待値 = https://example.com/created", link.CanonicalURL)
		}
		if link.Domain != "example.com" {
			t.Errorf("link.Domain = %v、期待値 = example.com", link.Domain)
		}
		if link.Title != "Created Link" {
			t.Errorf("link.Title = %v、期待値 = Created Link", link.Title)
		}
		if link.ImageURL != "https://example.com/created.png" {
			t.Errorf("link.ImageURL = %v、期待値 = https://example.com/created.png", link.ImageURL)
		}

		// 作成したリンクはcanonical URLで取得できなければならない。
		found, err := repo.FindByCanonicalURL(ctx, "https://example.com/created")
		if err != nil {
			t.Fatalf("FindByCanonicalURL()のエラー = %v", err)
		}
		if found == nil || found.ID != link.ID {
			t.Errorf("FindByCanonicalURL() = %v、IDが%vのリンクを期待", found, link.ID)
		}
	})

	t.Run("image_urlが空でも作成できる", func(t *testing.T) {
		link, err := repo.Create(ctx, repository.CreateLinkInput{
			CanonicalURL: "https://example.com/no-image",
			Domain:       "example.com",
			Title:        "No Image",
			ImageURL:     "",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		if link.ImageURL != "" {
			t.Errorf("link.ImageURL = %v、空を期待", link.ImageURL)
		}
	})
}
