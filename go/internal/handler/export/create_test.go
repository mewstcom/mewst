package export_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/handler/export"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// noopJobInserterは何もenqueueせずにdispatcher.JobInserterを満たす。
// これにより、Riverクライアントを動かさずにハンドラーテストで開始経路を実行できる。
type noopJobInserter struct{}

func (noopJobInserter) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

func newCreateExportUsecase(db *sql.DB, inserter dispatcher.JobInserter, storageReady bool) *usecase.CreateExportUsecase {
	q := query.New(db)
	return usecase.NewCreateExportUsecase(
		db,
		repository.NewUserProfileRepository(q),
		repository.NewExportRepository(q),
		dispatcher.NewDispatcher(inserter),
		storageReady,
	)
}

// newCreateHandlerは共有テストDBに対して動くUseCaseを持つHandlerを
// 構築する。エクスポートの開始は自身のtransactionを開くため、テストは
// フィクスチャをアウターのtransactionに保持せずcommitする。
func newCreateHandler(t *testing.T, storageReady bool) *export.Handler {
	t.Helper()

	db := testutil.GetTestDB()
	cfg := testutil.NewTestConfig(t)
	q := query.New(db)

	return export.NewHandler(
		cfg,
		session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly),
		usecase.NewGetExportShowUsecase(
			repository.NewUserProfileRepository(q),
			repository.NewExportRepository(q),
			storageReady,
		),
		newCreateExportUsecase(db, noopJobInserter{}, storageReady),
	)
}

// committedOwnerはプロフィールを所有するユーザーで、開始のUseCase自身の
// transactionから見えるよう共有テストDBへcommitしてある。
type committedOwner struct {
	testutil.ProfileOwner
	db *sql.DB
}

func newCommittedOwner(t *testing.T) committedOwner {
	t.Helper()

	db := testutil.GetTestDB()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("前提データ用transactionの開始に失敗: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	owner := testutil.NewProfileOwner(t, tx)
	if err := tx.Commit(); err != nil {
		t.Fatalf("前提データのcommitに失敗: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM export_completion_notifications WHERE profile_id = $1", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM export_posts WHERE export_id IN (SELECT id FROM exports WHERE profile_id = $1)", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM exports WHERE profile_id = $1", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(owner.ActorID))
		_, _ = db.Exec("DELETE FROM user_profiles WHERE profile_id = $1", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM profiles WHERE id = $1", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", uuid.UUID(owner.UserID))
	})

	return committedOwner{ProfileOwner: owner, db: db}
}

// newCreateRequestはPOST /settings/exportのリクエストを組み立てる。context
// には本番でCSRF / RequireAuthミドルウェアが渡すものを載せる。フォームが持つのは
// CSRFトークンだけで、その検証はこのハンドラーではなくミドルウェアが行う。
func newCreateRequest(t *testing.T, owner committedOwner) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/export", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: owner.UserID})
	ctx = middleware.SetProfileToContext(ctx, &model.Profile{ID: owner.ProfileID, Atname: "alice"})
	ctx = middleware.SetActorToContext(ctx, &model.Actor{ID: owner.ActorID, UserID: owner.UserID, ProfileID: owner.ProfileID})

	return req.WithContext(ctx)
}

// readFlashはレスポンスが設定したflashを返す。設定していない場合はnilを
// 返す。クッキーはbase64エンコードしたJSONを運ぶため、読み手の次のリクエストと
// 同じ方法でデコードする。
func readFlash(t *testing.T, rr *httptest.ResponseRecorder) *session.FlashMessage {
	t.Helper()

	for _, c := range rr.Result().Cookies() {
		if c.Name != session.FlashCookieName || c.Value == "" {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(c.Value)
		if err != nil {
			t.Fatalf("フラッシュクッキーのデコードに失敗: %v", err)
		}
		var flash session.FlashMessage
		if err := json.Unmarshal(data, &flash); err != nil {
			t.Fatalf("フラッシュメッセージのJSONパースに失敗: %v", err)
		}
		return &flash
	}
	return nil
}

func countExports(t *testing.T, owner committedOwner) int {
	t.Helper()

	var count int
	if err := owner.db.QueryRow(
		"SELECT COUNT(*) FROM exports WHERE profile_id = $1",
		uuid.UUID(owner.ProfileID),
	).Scan(&count); err != nil {
		t.Fatalf("エクスポート件数の取得に失敗: %v", err)
	}
	return count
}

// TestCreate_Successは、読み手が開始ボタンを押した後に受け取るレスポンスを
// 固定する。エクスポートが存在し、その新しい状態を説明する画面へ成功メッセージと
// ともに戻される。
func TestCreate_Success(t *testing.T) {
	t.Parallel()

	owner := newCommittedOwner(t)
	h := newCreateHandler(t, true)

	rr := httptest.NewRecorder()
	h.Create(rr, newCreateRequest(t, owner))

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/settings/export" {
		t.Errorf("リダイレクト先が不正: 実測値 = %v、期待値 = /settings/export", location)
	}

	flash := readFlash(t, rr)
	if flash == nil {
		t.Fatal("フラッシュメッセージが設定されていません")
	}
	if flash.Type != session.FlashSuccess {
		t.Errorf("フラッシュの種別 = %v、期待値 = %v", flash.Type, session.FlashSuccess)
	}
	if flash.Message != "エクスポートを開始しました。完了したらメールでお知らせします。" {
		t.Errorf("フラッシュの本文 = %q", flash.Message)
	}

	var status string
	if err := owner.db.QueryRow(
		"SELECT status FROM exports WHERE profile_id = $1",
		uuid.UUID(owner.ProfileID),
	).Scan(&status); err != nil {
		t.Fatalf("作成されたエクスポートの取得に失敗: %v", err)
	}
	if status != string(model.ExportStatusQueued) {
		t.Errorf("作成されたエクスポートのstatus = %q、期待値 = %q", status, model.ExportStatusQueued)
	}
}

// TestCreate_AlreadyInProgressは、エクスポートの実行中に2回目を押しても、
// エラーページではなく同じ画面に警告付きで留まることを固定する。壊れたものは無く、
// リクエストが効かなかっただけであるため。
func TestCreate_AlreadyInProgress(t *testing.T) {
	t.Parallel()

	owner := newCommittedOwner(t)
	h := newCreateHandler(t, true)

	rr := httptest.NewRecorder()
	h.Create(rr, newCreateRequest(t, owner))
	if rr.Code != http.StatusFound {
		t.Fatalf("1回目のステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	rr = httptest.NewRecorder()
	h.Create(rr, newCreateRequest(t, owner))

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/settings/export" {
		t.Errorf("リダイレクト先が不正: 実測値 = %v、期待値 = /settings/export", location)
	}

	flash := readFlash(t, rr)
	if flash == nil {
		t.Fatal("フラッシュメッセージが設定されていません")
	}
	if flash.Type != session.FlashWarning {
		t.Errorf("フラッシュの種別 = %v、期待値 = %v", flash.Type, session.FlashWarning)
	}
	if flash.Message != "エクスポートは既に処理中です。完了までお待ちください。" {
		t.Errorf("フラッシュの本文 = %q", flash.Message)
	}

	if n := countExports(t, owner); n != 1 {
		t.Errorf("エクスポートが%d件ある、期待値 = 1", n)
	}
}

// TestCreate_Unavailableは、オブジェクトストレージの無いデプロイが、機能を
// 提供していないことを表すstatusで答え、何も書き込まないことを固定する。そこでは
// 開始ボタンを描画しないため、このリクエストは現在の画面から来たものではない。
func TestCreate_Unavailable(t *testing.T) {
	t.Parallel()

	owner := newCommittedOwner(t)
	h := newCreateHandler(t, false)

	rr := httptest.NewRecorder()
	h.Create(rr, newCreateRequest(t, owner))

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusServiceUnavailable)
	}
	if flash := readFlash(t, rr); flash != nil {
		t.Errorf("フラッシュメッセージが設定されている: %v", flash)
	}
	if n := countExports(t, owner); n != 0 {
		t.Errorf("エクスポートが%d件作られている、期待値 = 0", n)
	}
}

// TestCreate_OtherProfileは、対象プロフィールを所有していないユーザーの
// セッションがnot foundとして拒否されることを固定する。呼び出し側が既存の
// プロフィールと存在しないプロフィールを区別できる痕跡は残さない。
func TestCreate_OtherProfile(t *testing.T) {
	t.Parallel()

	owner := newCommittedOwner(t)
	other := newCommittedOwner(t)
	h := newCreateHandler(t, true)

	req := newCreateRequest(t, owner)
	// 対象プロフィールはそのままに、別ユーザーのセッションへ差し替える。盗まれた
	// リクエストや細工されたリクエストはこの形になる。
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: other.UserID})
	ctx = middleware.SetActorToContext(ctx, &model.Actor{ID: other.ActorID, UserID: other.UserID, ProfileID: other.ProfileID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Typeが不正: 実測値 = %v、期待値 = text/html", contentType)
	}
	if n := countExports(t, owner); n != 0 {
		t.Errorf("エクスポートが%d件作られている、期待値 = 0", n)
	}
	if n := countExports(t, other); n != 0 {
		t.Errorf("別ユーザーのエクスポートが%d件作られている、期待値 = 0", n)
	}
}

// TestCreate_WithoutSessionは、RequireAuthが供給するidentityの一部を欠く
// リクエストが、所有者を推測せず失敗することを固定する。ここへ到達するのは未ログイン
// の訪問者が来たからではなく、RequireAuth無しでルートを登録したことを意味する。
func TestCreate_WithoutSession(t *testing.T) {
	t.Parallel()

	owner := newCommittedOwner(t)
	h := newCreateHandler(t, true)

	tests := []struct {
		name   string
		mutate func(ctx context.Context) context.Context
	}{
		{
			name:   "ユーザーが無い",
			mutate: func(ctx context.Context) context.Context { return middleware.SetUserToContext(ctx, nil) },
		},
		{
			name:   "プロフィールが無い",
			mutate: func(ctx context.Context) context.Context { return middleware.SetProfileToContext(ctx, nil) },
		},
		{
			name:   "アクターが無い",
			mutate: func(ctx context.Context) context.Context { return middleware.SetActorToContext(ctx, nil) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := newCreateRequest(t, owner)
			req = req.WithContext(tt.mutate(req.Context()))

			rr := httptest.NewRecorder()
			h.Create(rr, req)

			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusInternalServerError)
			}
			if n := countExports(t, owner); n != 0 {
				t.Errorf("エクスポートが%d件作られている、期待値 = 0", n)
			}
		})
	}
}
