package usecase_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// newFetchLinkMetadataUsecaseは本テスト用にFetchLinkMetadataUsecaseを組み立てる。
// httptestサーバーはloopback (127.0.0.1) でリッスンするため、private hostブロックは
// 無効化する (ブロック自体の検証は専用のサブテストで行う)。
func newFetchLinkMetadataUsecase(linkRepo *repository.LinkRepository) *usecase.FetchLinkMetadataUsecase {
	return usecase.NewFetchLinkMetadataUsecase(
		validator.NewLinkDataFetcherValidator(),
		linkRepo,
		&http.Client{},
		false,
	)
}

func TestFetchLinkMetadataUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("OGP付きページから新規リンクを作成する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		var canonicalURL string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, `<html><head>
				<link rel="canonical" href=%q>
				<meta property="og:title" content="OGP Title">
				<meta property="og:image" content="https://example.com/og.png">
				<title>Page Title</title>
			</head><body></body></html>`, canonicalURL)
		}))
		defer server.Close()
		canonicalURL = server.URL + "/articles/1"

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		// 対象URLはcanonical URLと異なる (トラッキングパラメータ付きなど)。
		// リンクはcanonical URLで保存される必要がある。
		targetURL := server.URL + "/articles/1?utm_source=x"
		output, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: targetURL})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		link := output.Link
		if link == nil {
			t.Fatal("output.Link = nil、リンクを期待")
		}
		if link.CanonicalURL != canonicalURL {
			t.Errorf("link.CanonicalURL = %q、期待値 = %q", link.CanonicalURL, canonicalURL)
		}
		if link.Domain != "127.0.0.1" {
			t.Errorf("link.Domain = %q、期待値 = 127.0.0.1", link.Domain)
		}
		if link.Title != "OGP Title" {
			t.Errorf("link.Title = %q、期待値 = OGP Title", link.Title)
		}
		if link.ImageURL != "https://example.com/og.png" {
			t.Errorf("link.ImageURL = %q、期待値 = https://example.com/og.png", link.ImageURL)
		}

		// リンクが永続化されていることを確認する
		saved, err := linkRepo.FindByCanonicalURL(ctx, canonicalURL)
		if err != nil {
			t.Fatalf("FindByCanonicalURL()のエラー = %v", err)
		}
		if saved == nil {
			t.Fatal("作成されたリンクがDBに存在しません")
		}
		if saved.ID != link.ID {
			t.Errorf("saved.ID = %v、期待値 = %v", saved.ID, link.ID)
		}
	})

	t.Run("og:titleが無い場合は <title> にフォールバックする", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, `<html><head><title>Page Title</title></head><body></body></html>`)
		}))
		defer server.Close()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		output, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: server.URL + "/no-ogp"})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Link.Title != "Page Title" {
			t.Errorf("link.Title = %q、期待値 = Page Title", output.Link.Title)
		}
		// canonicalタグが無いためcanonical URLは対象URLにフォールバックする
		if output.Link.CanonicalURL != server.URL+"/no-ogp" {
			t.Errorf("link.CanonicalURL = %q、期待値 = %q", output.Link.CanonicalURL, server.URL+"/no-ogp")
		}
		// og:imageが無い場合は空文字列で保存される
		if output.Link.ImageURL != "" {
			t.Errorf("link.ImageURL = %q、空を期待", output.Link.ImageURL)
		}
	})

	t.Run("タイトルが一切無い場合はcanonical URLをタイトルにする", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, `<html><head></head><body>hello</body></html>`)
		}))
		defer server.Close()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		targetURL := server.URL + "/no-title"
		output, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: targetURL})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Link.Title != targetURL {
			t.Errorf("link.Title = %q、期待値 = %q", output.Link.Title, targetURL)
		}
	})

	t.Run("対象URLと一致する既存リンクは取得せずに再利用する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		var hits atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			_, _ = fmt.Fprint(w, `<html><head><title>Should Not Be Fetched</title></head></html>`)
		}))
		defer server.Close()

		targetURL := server.URL + "/existing"
		linkID := testutil.NewLinkBuilder(t, tx).
			WithCanonicalURL(targetURL).
			Build()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		output, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: targetURL})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Link.ID != linkID {
			t.Errorf("output.Link.ID = %v、期待値 = %v", output.Link.ID, linkID)
		}
		if got := hits.Load(); got != 0 {
			t.Errorf("HTTPリクエスト回数 = %d、期待値 = 0 (既存リンク再利用時は取得しない)", got)
		}
	})

	t.Run("canonicalタグのURLで見つかる既存リンクを再利用する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		var canonicalURL string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, `<html><head><link rel="canonical" href=%q><title>Existing</title></head></html>`, canonicalURL)
		}))
		defer server.Close()
		canonicalURL = server.URL + "/canonical-page"

		linkID := testutil.NewLinkBuilder(t, tx).
			WithCanonicalURL(canonicalURL).
			Build()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		// 対象URLは既存リンクとは別だが、canonicalタグが既存リンクを指す
		targetURL := server.URL + "/canonical-page?ref=timeline"
		output, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: targetURL})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Link.ID != linkID {
			t.Errorf("output.Link.ID = %v、期待値 = %v (既存リンクの再利用)", output.Link.ID, linkID)
		}

		// 対象URLでは新しいリンクが作成されていないことを確認する
		duplicated, err := linkRepo.FindByCanonicalURL(ctx, targetURL)
		if err != nil {
			t.Fatalf("FindByCanonicalURL()のエラー = %v", err)
		}
		if duplicated != nil {
			t.Error("対象URLで新しいリンクが作成されています (canonicalの既存リンクを再利用すべき)")
		}
	})

	t.Run("同一ドメインのリダイレクトを追跡する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		mux := http.NewServeMux()
		mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dest", http.StatusFound)
		})
		mux.HandleFunc("/dest", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, `<html><head><meta property="og:title" content="Redirected Page"></head></html>`)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		targetURL := server.URL + "/start"
		output, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: targetURL})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Link.Title != "Redirected Page" {
			t.Errorf("link.Title = %q、期待値 = Redirected Page", output.Link.Title)
		}
		// ページにcanonicalタグが無いため、canonical URLはリダイレクト先ではなく
		// 元の対象URLにフォールバックする (Railsも元のtarget_urlでパースする)。
		if output.Link.CanonicalURL != targetURL {
			t.Errorf("link.CanonicalURL = %q、期待値 = %q", output.Link.CanonicalURL, targetURL)
		}
	})

	t.Run("許可していないドメインへのリダイレクトは追跡せず取得エラーになる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ボディ無しでリダイレクトを返す。リダイレクトは追跡されず、空の302
			// ボディがHTMLとして扱われ、取得エラーになる。
			w.Header().Set("Location", "http://disallowed.invalid/")
			w.WriteHeader(http.StatusFound)
		}))
		defer server.Close()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		targetURL := server.URL + "/phishing"
		output, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: targetURL})
		if output != nil {
			t.Errorf("出力 = %v、期待値 = nil", output)
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("ValidationErrorが期待されましたがerr = %v", err)
		}
		if !ve.HasFieldError("target_url") {
			t.Error("target_urlフィールドのエラーが期待されましたが、ありません")
		}

		// リンクが作成されていないことを確認する
		link, err := linkRepo.FindByCanonicalURL(ctx, targetURL)
		if err != nil {
			t.Fatalf("FindByCanonicalURL()のエラー = %v", err)
		}
		if link != nil {
			t.Error("取得エラー時にリンクが作成されています")
		}
	})

	t.Run("接続エラー時は取得エラーになる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		// サーバーを即座に閉じて接続エラーを発生させる
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		targetURL := server.URL + "/down"
		server.Close()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		_, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: targetURL})
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("ValidationErrorが期待されましたがerr = %v", err)
		}
		if !ve.HasFieldError("target_url") {
			t.Error("target_urlフィールドのエラーが期待されましたが、ありません")
		}
	})

	t.Run("og:imageが不正なURLの場合は取得エラーになる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, `<html><head>
				<meta property="og:title" content="Broken Image">
				<meta property="og:image" content="not a url">
			</head></html>`)
		}))
		defer server.Close()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		// RailsのLinkFormはimage_urlをurl: {allow_blank: true} で検証する
		// ため、パース不能なog:imageは取得エラーとして表示される。
		_, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: server.URL + "/broken-image"})
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("ValidationErrorが期待されましたがerr = %v", err)
		}
		if !ve.HasFieldError("target_url") {
			t.Error("target_urlフィールドのエラーが期待されましたが、ありません")
		}
	})

	t.Run("対象URLが空の場合はバリデーションエラーになる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		uc := newFetchLinkMetadataUsecase(linkRepo)

		_, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: ""})
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("ValidationErrorが期待されましたがerr = %v", err)
		}
		if !ve.HasFieldError("target_url") {
			t.Error("target_urlフィールドのエラーが期待されましたが、ありません")
		}
	})

	t.Run("private hostへの取得をブロックすると取得エラーになる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, `<html><head><title>Internal</title></head></html>`)
		}))
		defer server.Close()

		linkRepo := repository.NewLinkRepository(testutil.QueriesWithTx(tx))
		// private hostブロックを有効化する (本番設定)。httptestはloopback
		// (127.0.0.1) でリッスンするため、SSRF防御として取得が拒否される。
		uc := usecase.NewFetchLinkMetadataUsecase(
			validator.NewLinkDataFetcherValidator(),
			linkRepo,
			&http.Client{},
			true,
		)

		_, err := uc.Execute(ctx, usecase.FetchLinkMetadataInput{TargetURL: server.URL})
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("ValidationErrorが期待されましたがerr = %v", err)
		}
		if !ve.HasFieldError("target_url") {
			t.Error("target_urlフィールドのエラーが期待されましたが、ありません")
		}

		// ブロックされたため、リンクは作成されていないこと
		link, err := linkRepo.FindByCanonicalURL(ctx, server.URL)
		if err != nil {
			t.Fatalf("FindByCanonicalURL()のエラー = %v", err)
		}
		if link != nil {
			t.Error("private hostへのブロック時にリンクが作成されています")
		}
	})
}
