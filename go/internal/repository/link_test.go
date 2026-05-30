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

	t.Run("canonical URL でリンクを取得できる", func(t *testing.T) {
		link, err := repo.FindByCanonicalURL(ctx, canonicalURL)
		if err != nil {
			t.Fatalf("FindByCanonicalURL() error = %v", err)
		}
		if link == nil {
			t.Fatal("FindByCanonicalURL() = nil, want link")
		}
		if link.ID != linkID {
			t.Errorf("link.ID = %v, want %v", link.ID, linkID)
		}
		if link.CanonicalURL != canonicalURL {
			t.Errorf("link.CanonicalURL = %v, want %v", link.CanonicalURL, canonicalURL)
		}
		if link.Domain != "example.com" {
			t.Errorf("link.Domain = %v, want example.com", link.Domain)
		}
		if link.Title != "Example Title" {
			t.Errorf("link.Title = %v, want Example Title", link.Title)
		}
		if link.ImageURL != "https://example.com/og.png" {
			t.Errorf("link.ImageURL = %v, want https://example.com/og.png", link.ImageURL)
		}
	})

	t.Run("存在しない canonical URL は nil を返す", func(t *testing.T) {
		link, err := repo.FindByCanonicalURL(ctx, "https://example.com/nonexistent")
		if err != nil {
			t.Errorf("FindByCanonicalURL() error = %v, want nil", err)
		}
		if link != nil {
			t.Errorf("FindByCanonicalURL() link = %v, want nil", link)
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
			t.Fatalf("Create() error = %v", err)
		}

		if link.CanonicalURL != "https://example.com/created" {
			t.Errorf("link.CanonicalURL = %v, want https://example.com/created", link.CanonicalURL)
		}
		if link.Domain != "example.com" {
			t.Errorf("link.Domain = %v, want example.com", link.Domain)
		}
		if link.Title != "Created Link" {
			t.Errorf("link.Title = %v, want Created Link", link.Title)
		}
		if link.ImageURL != "https://example.com/created.png" {
			t.Errorf("link.ImageURL = %v, want https://example.com/created.png", link.ImageURL)
		}

		// The created link must be retrievable by its canonical URL.
		// [Ja] 作成したリンクは canonical URL で取得できなければならない。
		found, err := repo.FindByCanonicalURL(ctx, "https://example.com/created")
		if err != nil {
			t.Fatalf("FindByCanonicalURL() error = %v", err)
		}
		if found == nil || found.ID != link.ID {
			t.Errorf("FindByCanonicalURL() = %v, want link with ID %v", found, link.ID)
		}
	})

	t.Run("image_url が空でも作成できる", func(t *testing.T) {
		link, err := repo.Create(ctx, repository.CreateLinkInput{
			CanonicalURL: "https://example.com/no-image",
			Domain:       "example.com",
			Title:        "No Image",
			ImageURL:     "",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if link.ImageURL != "" {
			t.Errorf("link.ImageURL = %v, want empty", link.ImageURL)
		}
	})
}
