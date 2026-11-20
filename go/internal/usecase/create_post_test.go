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

// newCreatePostUsecaseは共有テストDBに対して動くCreatePostUsecaseを構築する。
// CreatePostUsecaseはdb.BeginTxで独自のトランザクションを開くため、前提となる行は
// アウターtxに保持したままでは内側のトランザクションから見えず、コミットしておく必要が
// ある。そのため各テストはGetTestDBを使い、Executeを呼ぶ前に前提データをコミットする。
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

	// サブテストは意図的に並列化しない: 各サブテストはuid = mewst-webの
	// oauth_applications行をコミットするが、uidにはUNIQUEインデックスがあるため、
	// 並列実行するとuidが衝突する。

	// 同じmewst-web行をコミットする他のテストパッケージ (handler/post等)
	// と直列化する。パッケージは共有DB上で別プロセスとして実行されるため、
	// 上記のパッケージ内直列化だけでは不十分。
	testutil.AcquireMewstWebLock(t)

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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if out == nil || out.Post == nil {
			t.Fatal("Postが返されていません")
		}
		post := out.Post

		// 投稿がpostsに作成されていること
		var (
			gotContent  string
			gotOauthID  uuid.UUID
			publishedAt time.Time
		)
		if err := db.QueryRow(
			`SELECT content, oauth_application_id, published_at FROM posts WHERE id = $1`,
			uuid.UUID(post.ID),
		).Scan(&gotContent, &gotOauthID, &publishedAt); err != nil {
			t.Fatalf("postsの取得に失敗: %v", err)
		}
		if gotContent != "Hello, Mewst!" {
			t.Errorf("content = %q、期待値 = %q", gotContent, "Hello, Mewst!")
		}
		if gotOauthID != uuid.UUID(post.OauthApplicationID) {
			t.Errorf("oauth_application_id = %v、期待値 = %v", gotOauthID, post.OauthApplicationID)
		}

		// 投稿者のlast_post_atが投稿のpublished_atで更新されていること
		var lastPostAt sql.NullTime
		if err := db.QueryRow(
			`SELECT last_post_at FROM profiles WHERE id = $1`, uuid.UUID(authorID),
		).Scan(&lastPostAt); err != nil {
			t.Fatalf("profilesの取得に失敗: %v", err)
		}
		if !lastPostAt.Valid || !lastPostAt.Time.Equal(post.PublishedAt) {
			t.Errorf("last_post_at = %v、期待値 = %v", lastPostAt, post.PublishedAt)
		}

		// 投稿者自身のホームタイムラインに追加されていること
		var timelineCount int
		if err := db.QueryRow(
			`SELECT count(*) FROM home_timeline_posts WHERE profile_id = $1 AND post_id = $2`,
			uuid.UUID(authorID), uuid.UUID(post.ID),
		).Scan(&timelineCount); err != nil {
			t.Fatalf("home_timeline_postsの取得に失敗: %v", err)
		}
		if timelineCount != 1 {
			t.Errorf("自身のホームタイムライン件数 = %d、期待値 = 1", timelineCount)
		}

		// コミット後にFanoutPostジョブが1件enqueueされていること
		if len(mock.inserts) != 1 {
			t.Fatalf("enqueue件数 = %d、期待値 = 1", len(mock.inserts))
		}
		args, ok := mock.inserts[0].(dispatcher.FanoutPostArgs)
		if !ok {
			t.Fatalf("argsの型がFanoutPostArgsではありません: %T", mock.inserts[0])
		}
		if args.PostID != post.ID.String() {
			t.Errorf("FanoutPostのPostID = %s、期待値 = %s", args.PostID, post.ID.String())
		}
	})

	t.Run("正常系: canonical_url指定でpost_linkを作成する", func(t *testing.T) {
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
			t.Fatalf("Execute()のエラー = %v", err)
		}

		// post_linksに該当linkとの関連付けが作成されていること
		var gotLinkID uuid.UUID
		if err := db.QueryRow(
			`SELECT link_id FROM post_links WHERE post_id = $1`, uuid.UUID(out.Post.ID),
		).Scan(&gotLinkID); err != nil {
			t.Fatalf("post_linksの取得に失敗: %v", err)
		}
		if gotLinkID != uuid.UUID(linkID) {
			t.Errorf("post_links.link_id = %v、期待値 = %v", gotLinkID, linkID)
		}
	})

	t.Run("正常系: 存在しないcanonical_urlではpost_linkを作成しない", func(t *testing.T) {
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

		// 一致するlink行が無いcanonical_urlはエラーにならず、post_linkも作成
		// されないこと (投稿自体は作成される)。
		out, err := uc.Execute(ctx, usecase.CreatePostInput{
			AuthorProfileID: authorID,
			Content:         "リンクなし",
			CanonicalURL:    "https://example.com/no-such-link",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		var count int
		if err := db.QueryRow(
			`SELECT count(*) FROM post_links WHERE post_id = $1`, uuid.UUID(out.Post.ID),
		).Scan(&count); err != nil {
			t.Fatalf("post_linksの件数取得に失敗: %v", err)
		}
		if count != 0 {
			t.Errorf("post_links件数 = %d、期待値 = 0", count)
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
			t.Errorf("Postが返されています: %+v", out)
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("ValidationErrorが期待されましたが、得られたエラー = %v", err)
		}
		if !ve.HasFieldError("content") {
			t.Error("contentフィールドのエラーが期待されましたが、ありません")
		}

		// 投稿が作成されていないこと
		var count int
		if err := db.QueryRow(
			`SELECT count(*) FROM posts WHERE profile_id = $1`, uuid.UUID(authorID),
		).Scan(&count); err != nil {
			t.Fatalf("postsの件数取得に失敗: %v", err)
		}
		if count != 0 {
			t.Errorf("posts件数 = %d、期待値 = 0", count)
		}
		// fanoutジョブもenqueueされていないこと
		if len(mock.inserts) != 0 {
			t.Errorf("enqueue件数 = %d、期待値 = 0", len(mock.inserts))
		}
	})

	t.Run("異常系: mewst-web OAuthアプリケーションが存在しないとAppErrorを返す", func(t *testing.T) {
		db := testutil.GetTestDB()
		ctx := context.Background()

		// mewst-webのoauth_applications行をあえて作らず、UseCaseが
		// AppErrCodeInternalを返すことを検証する。
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
			t.Errorf("Postが返されています: %+v", out)
		}
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("AppErrorが期待されましたが、得られたエラー = %v", err)
		}
		if ae.Code != model.AppErrCodeInternal {
			t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, model.AppErrCodeInternal)
		}

		// 投稿が作成されていないこと
		var count int
		if err := db.QueryRow(
			`SELECT count(*) FROM posts WHERE profile_id = $1`, uuid.UUID(authorID),
		).Scan(&count); err != nil {
			t.Fatalf("postsの件数取得に失敗: %v", err)
		}
		if count != 0 {
			t.Errorf("posts件数 = %d、期待値 = 0", count)
		}
		// fanoutジョブもenqueueされていないこと
		if len(mock.inserts) != 0 {
			t.Errorf("enqueue件数 = %d、期待値 = 0", len(mock.inserts))
		}
	})
}

// cleanupCreatePostDataは、指定した投稿者についてCreatePostがコミットした行
// (およびmewst-webのoauth application) を削除し、共有テストDBにテスト間の状態が
// 蓄積しないようにする。削除はFKの依存順に行う。
func cleanupCreatePostData(db *sql.DB, authorID model.ProfileID) {
	id := uuid.UUID(authorID)
	_, _ = db.Exec(`DELETE FROM home_timeline_posts WHERE post_id IN (SELECT id FROM posts WHERE profile_id = $1)`, id)
	_, _ = db.Exec(`DELETE FROM post_links WHERE post_id IN (SELECT id FROM posts WHERE profile_id = $1)`, id)
	_, _ = db.Exec(`DELETE FROM posts WHERE profile_id = $1`, id)
	_, _ = db.Exec(`DELETE FROM profiles WHERE id = $1`, id)
	_, _ = db.Exec(`DELETE FROM oauth_applications WHERE uid = $1`, model.MewstWebUID)
}
