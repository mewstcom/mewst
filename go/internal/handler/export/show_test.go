package export_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/handler/export"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func newExportHandler(t *testing.T, tx *sql.Tx, storageReady bool) *export.Handler {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)
	getExportShowUC := usecase.NewGetExportShowUsecase(
		repository.NewUserProfileRepository(queries),
		repository.NewExportRepository(queries),
		storageReady,
	)

	return export.NewHandler(testutil.NewTestConfig(t), getExportShowUC)
}

// newShowRequest builds a GET /settings/export request whose context carries
// what the CSRF and RequireAuth middleware supply in production: the locale, the
// CSRF token the start form submits, and the signed-in user and profile.
//
// [Ja] newShowRequest は GET /settings/export のリクエストを組み立てる。context には
// 本番で CSRF / RequireAuth ミドルウェアが渡すもの (ロケール、開始フォームが送信する
// CSRF トークン、ログイン中のユーザーとプロフィール) を載せる。
func newShowRequest(t *testing.T, owner testutil.ProfileOwner) *http.Request {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: owner.UserID})
	ctx = middleware.SetProfileToContext(ctx, &model.Profile{ID: owner.ProfileID, Atname: "alice"})

	req := httptest.NewRequest(http.MethodGet, "/settings/export", nil)
	return req.WithContext(ctx)
}

func assertContains(t *testing.T, body string, wants []string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていません", want)
		}
	}
}

func assertNotContains(t *testing.T, body string, unwants []string) {
	t.Helper()

	for _, unwant := range unwants {
		if strings.Contains(body, unwant) {
			t.Errorf("レスポンスに %q が含まれています", unwant)
		}
	}
}

// TestShow_WithoutExport pins the page a profile sees before it has ever
// requested an export: the description, the state in text, and the start form.
//
// [Ja] TestShow_WithoutExport は、一度もエクスポートを申請していないプロフィールが
// 見る画面を固定する。説明、テキストによる状態、開始フォームを含む。
func TestShow_WithoutExport(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	h := newExportHandler(t, tx, true)

	rr := httptest.NewRecorder()
	h.Show(rr, newShowRequest(t, owner))

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}
	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type が不正: got %v, want text/html", contentType)
	}

	body := rr.Body.String()
	assertContains(t, body, []string{
		"<h1",
		"エクスポート",
		"あなたのポストを月ごとの HTML ファイルにまとめた zip ファイルを作成します。作成には時間がかかることがあります。",
		"まだエクスポートを作成していません。",
		// The state is announced as a live region so a future in-place update is
		// conveyed to assistive technology.
		//
		// [Ja] 状態はライブリージョンとして通知し、将来その場で更新するようになった
		// ときに支援技術へ伝わるようにする。
		`role="status"`,
		// The start action is a native submit button inside a form that carries
		// the CSRF token, not a link or a JavaScript-driven control.
		//
		// [Ja] 開始操作はリンクや JavaScript 駆動の部品ではなく、CSRF トークンを
		// 持つフォーム内のネイティブな submit ボタン。
		`action="/settings/export"`,
		`method="POST"`,
		`name="csrf_token"`,
		`value="test-csrf-token"`,
		`<button class="btn rounded-full" type="submit">`,
		"エクスポートする",
		// The back affordance returns to the settings menu this page hangs off.
		//
		// [Ja] 戻る導線は、このページがぶら下がっている設定メニューへ戻る。
		`href="/settings"`,
	})
	assertNotContains(t, body, []string{`href="/settings/export/download"`})
}

// TestShow_English pins that the page is localized end to end: the title in the
// document head, the description, the state message, and both action labels
// come from the request's locale rather than from the Japanese default.
//
// [Ja] TestShow_English は画面が端から端まで国際化されていることを固定する。文書
// ヘッドのタイトル、説明、状態メッセージ、2 つの操作ラベルのいずれも、日本語の
// 既定ではなくリクエストのロケールから来る。
func TestShow_English(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	h := newExportHandler(t, tx, true)

	testutil.NewExportBuilder(t, tx).
		WithProfileID(owner.ProfileID).
		WithActorID(owner.ActorID).
		WithStatus(model.ExportStatusSucceeded).
		Build()

	req := newShowRequest(t, owner)
	req = req.WithContext(i18n.SetLocale(req.Context(), "en"))
	rr := httptest.NewRecorder()

	h.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	assertContains(t, body, []string{
		"<title>Export",
		"Your posts are collected into monthly HTML files in a zip file. Creating it may take a while.",
		"Your export is ready.",
		"Download your export",
		"Create an export",
	})
	assertNotContains(t, body, []string{"エクスポートの準備ができました。"})
}

// TestShow_ByState pins the state message and the actions offered for each
// state of the latest export, including the two-row cases where a newer export
// is in progress or failed while an earlier one still has a downloadable zip.
//
// [Ja] TestShow_ByState は、最新のエクスポートの状態ごとの状態メッセージと提供する
// 操作を固定する。より新しいエクスポートが進行中または失敗で、以前のものにまだ
// ダウンロード可能な zip がある 2 行のケースも含む。
func TestShow_ByState(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	const downloadHref = `href="/settings/export/download"`

	tests := []struct {
		name string
		// statuses are the profile's exports in creation order.
		//
		// [Ja] statuses は作成順に並べたプロフィールのエクスポート。
		statuses []model.ExportStatus
		wants    []string
		unwants  []string
	}{
		{
			name:     "進行中は完了をメールで知らせる旨を出し開始を出さない",
			statuses: []model.ExportStatus{model.ExportStatusStarted},
			wants:    []string{"エクスポートを作成しています。完了したらメールでお知らせします。"},
			unwants:  []string{"エクスポートする", downloadHref},
		},
		{
			name:     "成功はダウンロードリンクと再エクスポートの両方を出す",
			statuses: []model.ExportStatus{model.ExportStatusSucceeded},
			wants: []string{
				"エクスポートの準備ができました。",
				downloadHref,
				"エクスポートをダウンロードする",
				"エクスポートする",
			},
			unwants: []string{"以前のエクスポートをダウンロードする"},
		},
		{
			name:     "失敗は失敗を伝えて再実行を出す",
			statuses: []model.ExportStatus{model.ExportStatusFailed},
			wants:    []string{"エクスポートの作成に失敗しました。もう一度お試しください。", "エクスポートする"},
			unwants:  []string{downloadHref},
		},
		{
			name:     "進行中でも以前の成功は以前のものと分かる文言でダウンロードできる",
			statuses: []model.ExportStatus{model.ExportStatusSucceeded, model.ExportStatusQueued},
			wants: []string{
				"エクスポートを作成しています。完了したらメールでお知らせします。",
				downloadHref,
				"以前のエクスポートをダウンロードする",
			},
			unwants: []string{"エクスポートする"},
		},
		{
			name:     "失敗しても以前の成功はダウンロードできる",
			statuses: []model.ExportStatus{model.ExportStatusSucceeded, model.ExportStatusFailed},
			wants: []string{
				"エクスポートの作成に失敗しました。もう一度お試しください。",
				downloadHref,
				"以前のエクスポートをダウンロードする",
				"エクスポートする",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			owner := testutil.NewProfileOwner(t, tx)
			h := newExportHandler(t, tx, true)

			for i, status := range tt.statuses {
				testutil.NewExportBuilder(t, tx).
					WithProfileID(owner.ProfileID).
					WithActorID(owner.ActorID).
					WithStatus(status).
					WithCreatedAt(base.Add(time.Duration(i) * time.Hour)).
					Build()
			}

			rr := httptest.NewRecorder()
			h.Show(rr, newShowRequest(t, owner))

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			assertContains(t, body, tt.wants)
			assertNotContains(t, body, tt.unwants)
		})
	}
}

// TestShow_StorageNotConfigured pins that a deployment without the object
// storage says so instead of offering actions that cannot complete, even for a
// profile whose earlier export succeeded.
//
// [Ja] TestShow_StorageNotConfigured は、オブジェクトストレージの無いデプロイが、
// 以前のエクスポートが成功しているプロフィールに対しても、完了し得ない操作を出さずに
// その旨を伝えることを固定する。
func TestShow_StorageNotConfigured(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	h := newExportHandler(t, tx, false)

	testutil.NewExportBuilder(t, tx).
		WithProfileID(owner.ProfileID).
		WithActorID(owner.ActorID).
		WithStatus(model.ExportStatusSucceeded).
		Build()

	rr := httptest.NewRecorder()
	h.Show(rr, newShowRequest(t, owner))

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusServiceUnavailable)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type が不正: got %v, want text/html; charset=utf-8", contentType)
	}

	body := rr.Body.String()
	assertContains(t, body, []string{"エクスポートは現在ご利用いただけません。"})
	assertNotContains(t, body, []string{"エクスポートする", `href="/settings/export/download"`})
}

// TestShow_OtherProfile pins that the page refuses a profile the signed-in user
// does not own. The refusal is the 404 page, so the response does not reveal
// that the profile exists or that it has exports.
//
// [Ja] TestShow_OtherProfile は、ログイン中ユーザーが所有していないプロフィールを
// 画面が拒否することを固定する。拒否は 404 ページで返すため、応答からそのプロフィールが
// 存在することや、エクスポートを持つことは分からない。
func TestShow_OtherProfile(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	other := testutil.NewProfileOwner(t, tx)
	h := newExportHandler(t, tx, true)

	testutil.NewExportBuilder(t, tx).
		WithProfileID(owner.ProfileID).
		WithActorID(owner.ActorID).
		WithStatus(model.ExportStatusSucceeded).
		WithObjectKey("exports/" + owner.ProfileID.String() + "/secret.zip").
		Build()

	// Put the other user and the owner's profile in the request context to verify
	// that the use case checks the current ownership relation instead of trusting
	// the profile supplied by middleware.
	//
	// [Ja] 別のユーザーと所有者のプロフィールを request Context に組み合わせて入れ、
	// Usecase がミドルウェアから渡されたプロフィールを信頼せず、現在の所有関係を
	// 検証することを確認する。
	req := newShowRequest(t, testutil.ProfileOwner{UserID: other.UserID, ProfileID: owner.ProfileID})
	rr := httptest.NewRecorder()
	h.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusNotFound)
	}

	body := rr.Body.String()
	assertNotContains(t, body, []string{
		"エクスポートの準備ができました。",
		`href="/settings/export/download"`,
		"secret.zip",
	})
}

// TestShow_WithoutSession pins that the handler refuses to read exports when the
// context is missing either the signed-in user or the profile, which can only
// happen if the route is wired without RequireAuth. The user check
// short-circuits the guard, so a profile missing on its own needs its own case.
//
// The injected user carries no ID because the guard refuses before any export is
// read, so no row has to exist for it.
//
// [Ja] TestShow_WithoutSession は、context にログイン中ユーザーとプロフィールの
// どちらかが無いとき、ハンドラーがエクスポートを読まずに失敗することを固定する。
// これは RequireAuth 無しでルートを登録した場合にだけ起こりうる。ユーザーの判定で
// ガードが短絡するため、プロフィールだけが欠けた場合は別のケースとして与える。
//
// 注入するユーザーが ID を持たないのは、ガードがエクスポートを読む前に拒否するため、
// 対応する行が存在する必要が無いからである。
func TestShow_WithoutSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(ctx context.Context) context.Context
	}{
		{
			name:  "ユーザーもプロフィールも無い",
			setup: func(ctx context.Context) context.Context { return ctx },
		},
		{
			name: "プロフィールだけが無い",
			setup: func(ctx context.Context) context.Context {
				return middleware.SetUserToContext(ctx, &model.User{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			h := newExportHandler(t, tx, true)

			ctx := tt.setup(i18n.SetLocale(context.Background(), "ja"))
			req := httptest.NewRequest(http.MethodGet, "/settings/export", nil).WithContext(ctx)
			rr := httptest.NewRecorder()

			h.Show(rr, req)

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusInternalServerError)
			}
		})
	}
}
