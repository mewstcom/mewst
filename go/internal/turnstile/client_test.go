package turnstile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVerify_Success(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content-Typeがapplication/jsonであることを確認
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %v、期待値 = application/json", r.Header.Get("Content-Type"))
		}

		// 成功レスポンスを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"challenge_ts": "2024-01-01T00:00:00Z",
			"hostname": "mewst.com"
		}`))
	}))
	defer mockServer.Close()

	// クライアントを作成 (モックサーバーのURLを使用)
	client := NewClient("test-secret-key")
	// siteverifyURLを書き換えることができないため、httpClientのTransportを使ってリダイレクト
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "test-token")

	// アサーション
	if err != nil {
		t.Errorf("Verify()のエラー = %v、期待値 = nil", err)
	}
	if !success {
		t.Errorf("Verify()のsuccess = %v、期待値 = true", success)
	}
}

func TestVerify_Failure(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 失敗レスポンスを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": false}`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "invalid-token")

	// アサーション (検証失敗の場合はエラーが返る)
	if err == nil {
		t.Error("Verify()のエラー = nil、エラーを期待")
	}
	if success {
		t.Errorf("Verify()のsuccess = %v、期待値 = false", success)
	}
	if !strings.Contains(err.Error(), "turnstile検証に失敗しました") {
		t.Errorf("Verify()のエラー = %v、'turnstile検証に失敗しました'を含むエラーを期待", err)
	}
}

func TestVerify_FailureWithErrorCodes(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// エラーコード付きの失敗レスポンスを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": false,
			"error-codes": ["invalid-input-response", "timeout-or-duplicate"]
		}`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "invalid-token")

	// アサーション
	if err == nil {
		t.Error("Verify()のエラー = nil、エラーを期待")
	}
	if success {
		t.Errorf("Verify()のsuccess = %v、期待値 = false", success)
	}
	if !strings.Contains(err.Error(), "エラーコード") {
		t.Errorf("Verify()のエラー = %v、'エラーコード'を含むエラーを期待", err)
	}
}

func TestVerify_EmptyToken(t *testing.T) {
	t.Parallel()

	// クライアントを作成
	client := NewClient("test-secret-key")

	// Verifyメソッドをテスト (空トークン)
	ctx := context.Background()
	success, err := client.Verify(ctx, "")

	// アサーション
	if err == nil {
		t.Error("Verify()のエラー = nil、エラーを期待")
	}
	if success {
		t.Errorf("Verify()のsuccess = %v、期待値 = false", success)
	}
	if !strings.Contains(err.Error(), "トークンが空です") {
		t.Errorf("Verify()のエラー = %v、'トークンが空です'を含むエラーを期待", err)
	}
}

func TestVerify_EmptySecretKey(t *testing.T) {
	t.Parallel()

	// SecretKeyが空のクライアントを作成 (テスト環境用)
	client := NewClient("")

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "any-token")

	// アサーション (SecretKeyが空の場合は常に成功を返す)
	if err != nil {
		t.Errorf("Verify()のエラー = %v、期待値 = nil", err)
	}
	if !success {
		t.Errorf("Verify()のsuccess = %v、期待値 = true", success)
	}
}

func TestVerify_Timeout(t *testing.T) {
	t.Parallel()

	// コンテキストのタイムアウトを使ってタイムアウトをテスト
	// (HTTPクライアントのタイムアウト10秒より短いコンテキストを使用)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 2秒待機 (コンテキストのタイムアウト1秒より長い)
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// 1秒でタイムアウトするコンテキストを使用
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	success, err := client.Verify(ctx, "test-token")

	// アサーション
	if err == nil {
		t.Error("Verify()のエラー = nil、タイムアウトのエラーを期待")
	}
	if success {
		t.Errorf("Verify()のsuccess = %v、期待値 = false", success)
	}
}

func TestVerify_InvalidJSON(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 無効なJSONを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": invalid json`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "test-token")

	// アサーション
	if err == nil {
		t.Error("Verify()のエラー = nil、JSONのデコードエラーを期待")
	}
	if success {
		t.Errorf("Verify()のsuccess = %v、期待値 = false", success)
	}
	if !strings.Contains(err.Error(), "JSONデコードに失敗しました") {
		t.Errorf("Verify()のエラー = %v、'JSONデコードに失敗しました'を含むエラーを期待", err)
	}
}

func TestVerify_NonOKStatusCode(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 500エラーを返す
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Internal Server Error`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "test-token")

	// アサーション
	if err == nil {
		t.Error("Verify()のエラー = nil、HTTPのエラーを期待")
	}
	if success {
		t.Errorf("Verify()のsuccess = %v、期待値 = false", success)
	}
	if !strings.Contains(err.Error(), "siteverify APIがエラーを返しました") {
		t.Errorf("Verify()のエラー = %v、'siteverify APIがエラーを返しました'を含むエラーを期待", err)
	}
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	client := NewClient("my-secret-key")

	if client == nil {
		t.Fatal("NewClient() = nil、非nilを期待")
	}
	if client.secretKey != "my-secret-key" {
		t.Errorf("client.secretKey = %v、期待値 = my-secret-key", client.secretKey)
	}
	if client.httpClient == nil {
		t.Error("client.httpClient = nil、非nilを期待")
	}
	if client.httpClient.Timeout != requestTimeout {
		t.Errorf("client.httpClient.Timeout = %v、期待値 = %v", client.httpClient.Timeout, requestTimeout)
	}
}

// mockTransportはHTTPリクエストをモックサーバーにリダイレクトするTransport
type mockTransport struct {
	target string
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// リクエストURLをモックサーバーのURLに書き換え
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.target, "http://")
	req.URL.Path = ""
	return http.DefaultTransport.RoundTrip(req)
}
