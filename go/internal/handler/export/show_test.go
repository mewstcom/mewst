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
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// newExportHandlerは読み取り経路がテストのtransaction内で動くHandlerを
// 構築する。開始のUseCaseは配線の完全性のために渡すが、ここでは実行しない。
// 自身のtransactionを開くためこのtransactionが見えず、それを動かすテストは
// 代わりにフィクスチャをcommitするからである (create_test.goを参照)。
func newExportHandler(t *testing.T, tx *sql.Tx, storageReady bool) *export.Handler {
	t.Helper()

	cfg := testutil.NewTestConfig(t)
	queries := testutil.QueriesWithTx(tx)
	getExportShowUC := usecase.NewGetExportShowUsecase(
		repository.NewUserProfileRepository(queries),
		repository.NewExportRepository(queries),
		storageReady,
	)

	return export.NewHandler(
		cfg,
		session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly),
		getExportShowUC,
		newCreateExportUsecase(testutil.GetTestDB(), noopJobInserter{}, storageReady),
	)
}

// newShowRequestはGET /settings/exportのリクエストを組み立てる。contextには
// 本番でCSRF / RequireAuthミドルウェアが渡すもの (ロケール、開始フォームが送信する
// CSRFトークン、ログイン中のユーザーとプロフィール) を載せる。
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
			t.Errorf("レスポンスに%qが含まれていません", want)
		}
	}
}

func assertNotContains(t *testing.T, body string, unwants []string) {
	t.Helper()

	for _, unwant := range unwants {
		if strings.Contains(body, unwant) {
			t.Errorf("レスポンスに%qが含まれています", unwant)
		}
	}
}

// TestShow_WithoutExportは、一度もエクスポートを申請していないプロフィールが
// 見る画面を固定する。説明、テキストによる状態、開始フォームを含む。
func TestShow_WithoutExport(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	h := newExportHandler(t, tx, true)

	rr := httptest.NewRecorder()
	h.Show(rr, newShowRequest(t, owner))

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
	}
	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Typeが不正: 実測値 = %v、期待値 = text/html", contentType)
	}

	body := rr.Body.String()
	assertContains(t, body, []string{
		"<h1",
		"エクスポート",
		"あなたのポストを月ごとのHTMLファイルにまとめたzipファイルを作成します。作成には時間がかかることがあります。",
		"まだエクスポートを作成していません。",
		// 状態はライブリージョンとして通知し、将来その場で更新するようになった
		// ときに支援技術へ伝わるようにする。
		`role="status"`,
		// 開始操作はリンクやJavaScript駆動の部品ではなく、CSRFトークンを
		// 持つフォーム内のネイティブなsubmitボタン。
		`action="/settings/export"`,
		`method="POST"`,
		`name="csrf_token"`,
		`value="test-csrf-token"`,
		`<button class="btn rounded-full" type="submit">`,
		"エクスポートする",
		// 戻る導線は、このページがぶら下がっている設定メニューへ戻る。
		`href="/settings"`,
	})
	assertNotContains(t, body, []string{`href="/settings/export/download"`})
}

// TestShow_Englishは画面が端から端まで国際化されていることを固定する。文書
// ヘッドのタイトル、説明、状態メッセージ、2つの操作ラベルのいずれも、日本語の
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
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
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

// TestShow_ByStateは、最新のエクスポートの状態ごとの状態メッセージと提供する
// 操作を固定する。より新しいエクスポートが進行中または失敗で、以前のものにまだ
// ダウンロード可能なzipがある2行のケースも含む。
func TestShow_ByState(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	const downloadHref = `href="/settings/export/download"`

	tests := []struct {
		name string
		// statusesは作成順に並べたプロフィールのエクスポート。
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
				t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			assertContains(t, body, tt.wants)
			assertNotContains(t, body, tt.unwants)
		})
	}
}

// TestShow_StorageNotConfiguredは、オブジェクトストレージの無いデプロイが、
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
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusServiceUnavailable)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Typeが不正: 実測値 = %v、期待値 = text/html; charset=utf-8", contentType)
	}

	body := rr.Body.String()
	assertContains(t, body, []string{"エクスポートは現在ご利用いただけません。"})
	assertNotContains(t, body, []string{"エクスポートする", `href="/settings/export/download"`})
}

// TestShow_OtherProfileは、ログイン中ユーザーが所有していないプロフィールを
// 画面が拒否することを固定する。拒否は404ページで返すため、応答からそのプロフィールが
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

	// 別のユーザーと所有者のプロフィールをrequest Contextに組み合わせて入れ、
	// Usecaseがミドルウェアから渡されたプロフィールを信頼せず、現在の所有関係を
	// 検証することを確認する。
	req := newShowRequest(t, testutil.ProfileOwner{UserID: other.UserID, ProfileID: owner.ProfileID})
	rr := httptest.NewRecorder()
	h.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}

	body := rr.Body.String()
	assertNotContains(t, body, []string{
		"エクスポートの準備ができました。",
		`href="/settings/export/download"`,
		"secret.zip",
	})
}

// TestShow_WithoutSessionは、contextにログイン中ユーザーとプロフィールの
// どちらかが無いとき、ハンドラーがエクスポートを読まずに失敗することを固定する。
// これはRequireAuth無しでルートを登録した場合にだけ起こりうる。ユーザーの判定で
// ガードが短絡するため、プロフィールだけが欠けた場合は別のケースとして与える。
//
// 注入するユーザーがIDを持たないのは、ガードがエクスポートを読む前に拒否するため、
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
				t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusInternalServerError)
			}
		})
	}
}
