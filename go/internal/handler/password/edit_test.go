package password_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestEdit_WithValidSucceededEmailConfirmation(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	// 確認済みのメール確認レコードを作成
	now := time.Now()
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		WithSucceededAt(now).
		Build()

	// CSRFトークンをコンテキストに設定
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	// レスポンスにパスワード入力フォームが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "csrf_token") {
		t.Error("CSRFトークンがフォームに含まれていません")
	}
	if !strings.Contains(body, `name="password"`) {
		t.Error("パスワードフィールドがフォームに含まれていません")
	}
	if !strings.Contains(body, `_method`) {
		t.Error("メソッドオーバーライドがフォームに含まれていません")
	}
}

func TestEdit_WithoutEmailConfirmationID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// クッキーなしでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestEdit_WithInvalidEmailConfirmationID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 無効なUUIDでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: "invalid-uuid",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestEdit_WithUnsuccessfulEmailConfirmation(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	// 未確認のメール確認レコードを作成(succeeded_atがNULL)
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestEdit_WithMismatchedEvent(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	// password_reset 以外のイベント(sign_up)の確認済みレコードを作成
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("sign_up").
		WithSucceededAt(time.Now()).
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// イベント種別が異なるためルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}
