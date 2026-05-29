package email

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates/emails/email_confirmation"
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
