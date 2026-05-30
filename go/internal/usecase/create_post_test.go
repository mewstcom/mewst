package usecase_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// newCreatePostUsecase builds a CreatePostUsecase whose repositories run against
// the shared test DB. CreatePostUsecase opens its own transaction via
// db.BeginTx, so prerequisite rows must be committed (not held in an outer tx)
// to be visible to that inner transaction; tests therefore use GetTestDB and
// commit their setup data before calling Execute.
//
// [Ja] newCreatePostUsecase は共有テスト DB に対して動く CreatePostUsecase を構築する。
// CreatePostUsecase は db.BeginTx で独自のトランザクションを開くため、前提となる行は
// アウター tx に保持したままでは内側のトランザクションから見えず、コミットしておく必要が
// ある。そのため各テストは GetTestDB を使い、Execute を呼ぶ前に前提データをコミットする。
func newCreatePostUsecase(db *sql.DB, mock *recordingJobInserter) *usecase.CreatePostUsecase {
	q := query.New(db)
	return usecase.NewCreatePostUsecase(
		db,
		validator.NewPostCreateValidator(),
		repository.NewOauthApplicationRepository(q),
		repository.NewLinkRepository(q),
		repository.NewPostRepository(q),
		repository.NewPostLinkRepository(q),
		repository.NewProfileRepository(q),
		repository.NewHomeTimelinePostRepository(q),
		dispatcher.NewDispatcher(mock),
	)
}

func TestCreatePostUsecase_Execute(t *testing.T) {
	t.Parallel()

	// Subtests are intentionally NOT parallel: each commits an oauth_applications
	// row with uid = mewst-web, which has a UNIQUE index. Running them in
	// parallel would collide on that uid.
	// [Ja] サブテストは意図的に並列化しない: 各サブテストは uid = mewst-web の
	// oauth_applications 行をコミットするが、uid には UNIQUE インデックスがあるため、
	// 並列実行すると uid が衝突する。

	t.Run("正常系: 投稿を作成し副作用が発生する", func(t *testing.T) {
		db := testutil.GetTestDB()
		ctx := context.Background()

		setupTx, err := db.Begin()
		if err != nil {
			t.Fatalf("セットアップ用トランザクションの開始に失敗: %v", err)
		}
		defer func() { _ = setupTx.Rollback() }()
		authorID := testutil.NewProfileBuilder(t, setupTx).Build()
		testutil.NewOauthApplicationBuilder(t, setupTx).WithUID(model.MewstWebUID).Build()
		if err := setupTx.Commit(); err != nil {
			t.Fatalf("前提データのコミットに失敗: %v", err)
		}
		t.Cleanup(func() { cleanupCreatePostData(db, authorID) })

		mock := &recordingJobInserter{}
		uc := newCreatePostUsecase(db, mock)

		out, err := uc.Execute(ctx, usecase.CreatePostInput{
			AuthorProfileID: authorID,
			Content:         "Hello, Mewst!",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if out == nil || out.Post == nil {
			t.Fatal("Post が返されていません")
		}
		post := out.Post

		// 投稿が posts に作成されていること
		var (
			gotContent  string
			gotOauthID  uuid.UUID
			publishedAt time.Time
		)
		if err := db.QueryRow(
			`SELECT content, oauth_application_id, published_at FROM posts WHERE id = $1`,
			uuid.UUID(post.ID),
		).Scan(&gotContent, &gotOauthID, &publishedAt); err != nil {
			t.Fatalf("posts の取得に失敗: %v", err)
		}
		if gotContent != "Hello, Mewst!" {
			t.Errorf("content = %q, want %q", gotContent, "Hello, Mewst!")
		}
		if gotOauthID != uuid.UUID(post.OauthApplicationID) {
			t.Errorf("oauth_application_id = %v, want %v", gotOauthID, post.OauthApplicationID)
		}

		// 投稿者の last_post_at が投稿の published_at で更新されていること
		var lastPostAt sql.NullTime
		if err := db.QueryRow(
			`SELECT last_post_at FROM profiles WHERE id = $1`, uuid.UUID(authorID),
		).Scan(&lastPostAt); err != nil {
			t.Fatalf("profiles の取得に失敗: %v", err)
		}
		if !lastPostAt.Valid || !lastPostAt.Time.Equal(post.PublishedAt) {
			t.Errorf("last_post_at = %v, want %v", lastPostAt, post.PublishedAt)
		}

		// 投稿者自身のホームタイムラインに追加されていること
		var timelineCount int
		if err := db.QueryRow(
			`SELECT count(*) FROM home_timeline_posts WHERE profile_id = $1 AND post_id = $2`,
			uuid.UUID(authorID), uuid.UUID(post.ID),
		).Scan(&timelineCount); err != nil {
			t.Fatalf("home_timeline_posts の取得に失敗: %v", err)
		}
		if timelineCount != 1 {
			t.Errorf("自身のホームタイムライン件数 = %d, want 1", timelineCount)
		}

		// コミット後に FanoutPost ジョブが 1 件 enqueue されていること
		if len(mock.inserts) != 1 {
			t.Fatalf("enqueue 件数 = %d, want 1", len(mock.inserts))
		}
		args, ok := mock.inserts[0].(dispatcher.FanoutPostArgs)
		if !ok {
			t.Fatalf("args の型が FanoutPostArgs ではありません: %T", mock.inserts[0])
		}
		if args.PostID != post.ID.String() {
			t.Errorf("FanoutPost の PostID = %s, want %s", args.PostID, post.ID.String())
		}
	})

	t.Run("正常系: canonical_url 指定で post_link を作成する", func(t *testing.T) {
		db := testutil.GetTestDB()
		ctx := context.Background()

		setupTx, err := db.Begin()
		if err != nil {
			t.Fatalf("セットアップ用トランザクションの開始に失敗: %v", err)
		}
		defer func() { _ = setupTx.Rollback() }()
		authorID := testutil.NewProfileBuilder(t, setupTx).Build()
		testutil.NewOauthApplicationBuilder(t, setupTx).WithUID(model.MewstWebUID).Build()
		canonicalURL := "https://example.com/create-post-with-link"
		linkID := testutil.NewLinkBuilder(t, setupTx).WithCanonicalURL(canonicalURL).Build()
		if err := setupTx.Commit(); err != nil {
			t.Fatalf("前提データのコミットに失敗: %v", err)
		}
		t.Cleanup(func() {
			cleanupCreatePostData(db, authorID)
			_, _ = db.Exec(`DELETE FROM links WHERE id = $1`, uuid.UUID(linkID))
		})

		mock := &recordingJobInserter{}
		uc := newCreatePostUsecase(db, mock)

		out, err := uc.Execute(ctx, usecase.CreatePostInput{
			AuthorProfileID: authorID,
			Content:         "リンク付き投稿",
			CanonicalURL:    canonicalURL,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// post_links に該当 link との関連付けが作成されていること
		var gotLinkID uuid.UUID
		if err := db.QueryRow(
			`SELECT link_id FROM post_links WHERE post_id = $1`, uuid.UUID(out.Post.ID),
		).Scan(&gotLinkID); err != nil {
			t.Fatalf("post_links の取得に失敗: %v", err)
		}
		if gotLinkID != uuid.UUID(linkID) {
			t.Errorf("post_links.link_id = %v, want %v", gotLinkID, linkID)
		}
	})

	t.Run("正常系: 存在しない canonical_url では post_link を作成しない", func(t *testing.T) {
		db := testutil.GetTestDB()
		ctx := context.Background()

		setupTx, err := db.Begin()
		if err != nil {
			t.Fatalf("セットアップ用トランザクションの開始に失敗: %v", err)
		}
		defer func() { _ = setupTx.Rollback() }()
		authorID := testutil.NewProfileBuilder(t, setupTx).Build()
		testutil.NewOauthApplicationBuilder(t, setupTx).WithUID(model.MewstWebUID).Build()
		if err := setupTx.Commit(); err != nil {
			t.Fatalf("前提データのコミットに失敗: %v", err)
		}
		t.Cleanup(func() { cleanupCreatePostData(db, authorID) })

		mock := &recordingJobInserter{}
		uc := newCreatePostUsecase(db, mock)

		// A canonical_url with no matching link row must not error and must not
		// create a post_link (the post is still created).
		// [Ja] 一致する link 行が無い canonical_url はエラーにならず、post_link も作成
		// されないこと (投稿自体は作成される)。
		out, err := uc.Execute(ctx, usecase.CreatePostInput{
			AuthorProfileID: authorID,
			Content:         "リンクなし",
			CanonicalURL:    "https://example.com/no-such-link",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		var count int
		if err := db.QueryRow(
			`SELECT count(*) FROM post_links WHERE post_id = $1`, uuid.UUID(out.Post.ID),
		).Scan(&count); err != nil {
			t.Fatalf("post_links の件数取得に失敗: %v", err)
		}
		if count != 0 {
			t.Errorf("post_links 件数 = %d, want 0", count)
		}
	})

	t.Run("異常系: 本文が空ならバリデーションエラーで副作用が発生しない", func(t *testing.T) {
		db := testutil.GetTestDB()
		ctx := context.Background()

		setupTx, err := db.Begin()
		if err != nil {
			t.Fatalf("セットアップ用トランザクションの開始に失敗: %v", err)
		}
		defer func() { _ = setupTx.Rollback() }()
		authorID := testutil.NewProfileBuilder(t, setupTx).Build()
		if err := setupTx.Commit(); err != nil {
			t.Fatalf("前提データのコミットに失敗: %v", err)
		}
		t.Cleanup(func() { cleanupCreatePostData(db, authorID) })

		mock := &recordingJobInserter{}
		uc := newCreatePostUsecase(db, mock)

		out, err := uc.Execute(ctx, usecase.CreatePostInput{
			AuthorProfileID: authorID,
			Content:         "",
		})
		if out != nil {
			t.Errorf("Post が返されています: %+v", out)
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("ValidationError が期待されましたが、得られたエラー = %v", err)
		}
		if !ve.HasFieldError("content") {
			t.Error("content フィールドのエラーが期待されましたが、ありません")
		}

		// 投稿が作成されていないこと
		var count int
		if err := db.QueryRow(
			`SELECT count(*) FROM posts WHERE profile_id = $1`, uuid.UUID(authorID),
		).Scan(&count); err != nil {
			t.Fatalf("posts の件数取得に失敗: %v", err)
		}
		if count != 0 {
			t.Errorf("posts 件数 = %d, want 0", count)
		}
		// fanout ジョブも enqueue されていないこと
		if len(mock.inserts) != 0 {
			t.Errorf("enqueue 件数 = %d, want 0", len(mock.inserts))
		}
	})

	t.Run("異常系: mewst-web OAuth アプリケーションが存在しないと AppError を返す", func(t *testing.T) {
		db := testutil.GetTestDB()
		ctx := context.Background()

		// Intentionally skip creating the mewst-web oauth_applications row to
		// verify the UseCase returns AppErrCodeInternal when it is absent.
		//
		// [Ja] mewst-web の oauth_applications 行をあえて作らず、UseCase が
		// AppErrCodeInternal を返すことを検証する。
		setupTx, err := db.Begin()
		if err != nil {
			t.Fatalf("セットアップ用トランザクションの開始に失敗: %v", err)
		}
		defer func() { _ = setupTx.Rollback() }()
		authorID := testutil.NewProfileBuilder(t, setupTx).Build()
		if err := setupTx.Commit(); err != nil {
			t.Fatalf("前提データのコミットに失敗: %v", err)
		}
		t.Cleanup(func() { cleanupCreatePostData(db, authorID) })

		mock := &recordingJobInserter{}
		uc := newCreatePostUsecase(db, mock)

		out, err := uc.Execute(ctx, usecase.CreatePostInput{
			AuthorProfileID: authorID,
			Content:         "mewst-web 不在",
		})
		if out != nil {
			t.Errorf("Post が返されています: %+v", out)
		}
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("AppError が期待されましたが、得られたエラー = %v", err)
		}
		if ae.Code != model.AppErrCodeInternal {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeInternal)
		}

		// 投稿が作成されていないこと
		var count int
		if err := db.QueryRow(
			`SELECT count(*) FROM posts WHERE profile_id = $1`, uuid.UUID(authorID),
		).Scan(&count); err != nil {
			t.Fatalf("posts の件数取得に失敗: %v", err)
		}
		if count != 0 {
			t.Errorf("posts 件数 = %d, want 0", count)
		}
		// fanout ジョブも enqueue されていないこと
		if len(mock.inserts) != 0 {
			t.Errorf("enqueue 件数 = %d, want 0", len(mock.inserts))
		}
	})
}

// cleanupCreatePostData removes the rows a CreatePost run commits for the given
// author (and its mewst-web oauth application) so the shared test DB does not
// accumulate state across tests. Deletions follow FK dependency order.
//
// [Ja] cleanupCreatePostData は、指定した投稿者について CreatePost がコミットした行
// (および mewst-web の oauth application) を削除し、共有テスト DB にテスト間の状態が
// 蓄積しないようにする。削除は FK の依存順に行う。
func cleanupCreatePostData(db *sql.DB, authorID model.ProfileID) {
	id := uuid.UUID(authorID)
	_, _ = db.Exec(`DELETE FROM home_timeline_posts WHERE post_id IN (SELECT id FROM posts WHERE profile_id = $1)`, id)
	_, _ = db.Exec(`DELETE FROM post_links WHERE post_id IN (SELECT id FROM posts WHERE profile_id = $1)`, id)
	_, _ = db.Exec(`DELETE FROM posts WHERE profile_id = $1`, id)
	_, _ = db.Exec(`DELETE FROM profiles WHERE id = $1`, id)
	_, _ = db.Exec(`DELETE FROM oauth_applications WHERE uid = $1`, model.MewstWebUID)
}
