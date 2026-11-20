package usecase_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestGetLinkUsecase_Execute_Found(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	linkID := testutil.NewLinkBuilder(t, tx).
		WithCanonicalURL("https://example.com/get-link-found").
		WithTitle("Found Link").
		Build()

	linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewGetLinkUsecase(linkRepo)
	result, err := uc.Execute(ctx, usecase.GetLinkInput{
		CanonicalURL: "https://example.com/get-link-found",
	})

	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if result == nil || result.Link == nil {
		t.Fatal("Link = nil、非nilを期待")
	}
	if result.Link.ID != linkID {
		t.Errorf("ID = %v、期待値 = %v", result.Link.ID, linkID)
	}
	if result.Link.Title != "Found Link" {
		t.Errorf("Title = %v、期待値 = %v", result.Link.Title, "Found Link")
	}
}

func TestGetLinkUsecase_Execute_NotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// 未知のURLはエラーではなくLink = nilとして返り、呼び出し側が
	// フォールバックできること。
	linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewGetLinkUsecase(linkRepo)
	result, err := uc.Execute(ctx, usecase.GetLinkInput{
		CanonicalURL: "https://example.com/get-link-not-found",
	})

	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if result == nil {
		t.Fatal("Execute()の結果 = nil、非nilを期待")
	}
	if result.Link != nil {
		t.Errorf("Link = %+v、期待値 = nil", result.Link)
	}
}
