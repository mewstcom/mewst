package email

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates/emails/email_confirmation"
	"github.com/mewstcom/mewst/go/internal/templates/emails/export_completed"
)

func TestResendSender_from_WithName(t *testing.T) {
	t.Parallel()

	sender := NewResendSender("dummy-api-key", "noreply@example.com", "Mewst")

	got := sender.from()
	want := "Mewst <noreply@example.com>"

	if got != want {
		t.Errorf("from() = %q, want %q", got, want)
	}
}

func TestResendSender_from_WithoutName(t *testing.T) {
	t.Parallel()

	sender := NewResendSender("dummy-api-key", "noreply@example.com", "")

	got := sender.from()
	want := "noreply@example.com"

	if got != want {
		t.Errorf("from() = %q, want %q", got, want)
	}
}

func TestDiscardSender_SendConcurrently(t *testing.T) {
	t.Parallel()

	sender := NewDiscardSender()
	ctx := context.Background()

	const sendCount = 32
	errCh := make(chan error, sendCount)
	var wg sync.WaitGroup

	for i := 0; i < sendCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- sender.Send(ctx, SendInput{
				To:      "test@example.com",
				Subject: "テスト件名",
			})
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Errorf("Send failed: %v", err)
		}
	}
}

func TestNoopSender_Send(t *testing.T) {
	t.Parallel()

	sender := NewNoopSender()
	ctx := context.Background()

	input := SendInput{
		To:       "test@example.com",
		Subject:  "テスト件名",
		HTMLBody: email_confirmation.JaHTML("test@example.com", "123456"),
		TextBody: email_confirmation.JaText("test@example.com", "123456"),
	}

	err := sender.Send(ctx, input)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// 送信されたメールが記録されているか確認
	if len(sender.SentEmails) != 1 {
		t.Errorf("expected 1 sent email, got %d", len(sender.SentEmails))
	}

	if sender.SentEmails[0].To != "test@example.com" {
		t.Errorf("expected to=test@example.com, got %s", sender.SentEmails[0].To)
	}

	if sender.SentEmails[0].Subject != "テスト件名" {
		t.Errorf("expected subject=テスト件名, got %s", sender.SentEmails[0].Subject)
	}
}

func TestNoopSender_MultipleSends(t *testing.T) {
	t.Parallel()

	sender := NewNoopSender()
	ctx := context.Background()

	// 複数回メール送信
	for i := 0; i < 3; i++ {
		input := SendInput{
			To:       "test@example.com",
			Subject:  "テスト",
			HTMLBody: email_confirmation.JaHTML("test@example.com", "123456"),
			TextBody: email_confirmation.JaText("test@example.com", "123456"),
		}
		err := sender.Send(ctx, input)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	if len(sender.SentEmails) != 3 {
		t.Errorf("expected 3 sent emails, got %d", len(sender.SentEmails))
	}
}

func TestEmailConfirmationTemplate_Japanese_HTML(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// テンプレートをレンダリング
	var buf bytes.Buffer
	err := email_confirmation.JaHTML("user@example.com", "654321").Render(ctx, &buf)
	if err != nil {
		t.Fatalf("JaHTML render failed: %v", err)
	}

	html := buf.String()

	// 確認コードが含まれているか
	if !strings.Contains(html, "654321") {
		t.Error("expected confirmation code in HTML")
	}

	// HTMLタグが含まれているか (templは小文字に変換する)
	if !strings.Contains(html, "<!doctype html>") {
		t.Error("expected doctype in HTML")
	}

	// lang属性が日本語になっているか
	if !strings.Contains(html, `lang="ja"`) {
		t.Error("expected lang=ja in HTML")
	}

	// メールアドレスが含まれているか
	if !strings.Contains(html, "user@example.com") {
		t.Error("expected email address in HTML")
	}
}

func TestEmailConfirmationTemplate_Japanese_Text(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// テンプレートをレンダリング
	var buf bytes.Buffer
	err := email_confirmation.JaText("user@example.com", "654321").Render(ctx, &buf)
	if err != nil {
		t.Fatalf("JaText render failed: %v", err)
	}

	text := buf.String()

	// 確認コードが含まれているか
	if !strings.Contains(text, "654321") {
		t.Error("expected confirmation code in text")
	}

	// メールアドレスが含まれているか
	if !strings.Contains(text, "user@example.com") {
		t.Error("expected email address in text")
	}

	// 日本語メッセージが含まれているか
	if !strings.Contains(text, "確認用コード") {
		t.Error("expected Japanese message in text")
	}
}

func TestEmailConfirmationTemplate_English_HTML(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// テンプレートをレンダリング
	var buf bytes.Buffer
	err := email_confirmation.EnHTML("user@example.com", "987654").Render(ctx, &buf)
	if err != nil {
		t.Fatalf("EnHTML render failed: %v", err)
	}

	html := buf.String()

	// 確認コードが含まれているか
	if !strings.Contains(html, "987654") {
		t.Error("expected confirmation code in HTML")
	}

	// lang属性が英語になっているか
	if !strings.Contains(html, `lang="en"`) {
		t.Error("expected lang=en in HTML")
	}

	// 英語メッセージが含まれているか
	if !strings.Contains(html, "confirmation code") {
		t.Error("expected English message in HTML")
	}
}

func TestEmailConfirmationTemplate_English_Text(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// テンプレートをレンダリング
	var buf bytes.Buffer
	err := email_confirmation.EnText("user@example.com", "987654").Render(ctx, &buf)
	if err != nil {
		t.Fatalf("EnText render failed: %v", err)
	}

	text := buf.String()

	// 確認コードが含まれているか
	if !strings.Contains(text, "987654") {
		t.Error("expected confirmation code in text")
	}

	// 英語メッセージが含まれているか
	if !strings.Contains(text, "confirmation code") {
		t.Error("expected English message in text")
	}
}

// capturedRequest is what the fake Resend endpoint saw.
//
// [Ja] capturedRequest は偽の Resend エンドポイントが見たもの。
type capturedRequest struct {
	idempotencyKey        string
	idempotencyKeyPresent bool
	body                  map[string]any
}

// newTestResendSender points a ResendSender at a local endpoint that records
// the request, so that the header the SDK actually puts on the wire can be
// asserted rather than the field the caller set.
//
// [Ja] newTestResendSender は ResendSender をリクエストを記録するローカルの
// エンドポイントへ向ける。呼び出し元が設定したフィールドではなく、SDK が実際に
// 通信へ載せるヘッダーを検証するため。
func newTestResendSender(t *testing.T, captured *capturedRequest) *ResendSender {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.idempotencyKey = r.Header.Get("Idempotency-Key")
		_, captured.idempotencyKeyPresent = r.Header[http.CanonicalHeaderKey("Idempotency-Key")]

		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(rawBody, &captured.body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"id":"test-email-id"}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}

	sender := NewResendSender("dummy-api-key", "noreply@example.com", "Mewst")
	sender.client.BaseURL = baseURL

	return sender
}

func TestResendSender_Send_WithIdempotencyKey(t *testing.T) {
	t.Parallel()

	var captured capturedRequest
	sender := newTestResendSender(t, &captured)

	exportURL := "https://mewst.com/settings/export"
	err := sender.Send(context.Background(), SendInput{
		To:             "test@example.com",
		Subject:        "[Mewst] エクスポートの準備ができました",
		HTMLBody:       export_completed.JaHTML(exportURL),
		TextBody:       export_completed.JaText(exportURL),
		IdempotencyKey: "export-completed/01J000000000000000000EXPRT",
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if captured.idempotencyKey != "export-completed/01J000000000000000000EXPRT" {
		t.Errorf("Idempotency-Key header = %q, want %q", captured.idempotencyKey, "export-completed/01J000000000000000000EXPRT")
	}

	if html, _ := captured.body["html"].(string); !strings.Contains(html, exportURL) {
		t.Error("request body does not contain the rendered HTML body")
	}
	if text, _ := captured.body["text"].(string); !strings.Contains(text, exportURL) {
		t.Error("request body does not contain the rendered text body")
	}
}

// TestResendSender_Send_WithoutIdempotencyKey pins the compatibility of the
// mails that predate the key: they leave it unset, and the request must go out
// without the header rather than with an empty one, which the API would reject
// or treat as a shared key.
//
// [Ja] TestResendSender_Send_WithoutIdempotencyKey は、キー導入前からのメールの
// 互換性を固定する。それらはキーを設定しないため、リクエストは空のヘッダーでは
// なくヘッダー無しで出る必要がある (空のヘッダーは API に拒否されるか、共有された
// キーとして扱われる)。
func TestResendSender_Send_WithoutIdempotencyKey(t *testing.T) {
	t.Parallel()

	var captured capturedRequest
	sender := newTestResendSender(t, &captured)

	err := sender.Send(context.Background(), SendInput{
		To:       "test@example.com",
		Subject:  "[Mewst] 確認用コード",
		HTMLBody: email_confirmation.JaHTML("test@example.com", "123456"),
		TextBody: email_confirmation.JaText("test@example.com", "123456"),
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if captured.idempotencyKeyPresent {
		t.Errorf("Idempotency-Key header is present with value %q, want it to be absent", captured.idempotencyKey)
	}

	if from, _ := captured.body["from"].(string); from != "Mewst <noreply@example.com>" {
		t.Errorf("from = %q, want %q", from, "Mewst <noreply@example.com>")
	}
	if html, _ := captured.body["html"].(string); !strings.Contains(html, "123456") {
		t.Error("request body does not contain the rendered HTML body")
	}
}
