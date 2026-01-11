package email

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/internal/templates"
	"github.com/mewstcom/mewst/internal/templates/emails"
)

func TestNoopSender_SendEmailConfirmation(t *testing.T) {
	t.Parallel()

	sender := NewNoopSender()

	// コンテキストにロケールを設定
	ctx := templates.WithLocale(context.Background(), "ja")

	// テスト用のテンプレートデータ
	templateData := emails.EmailConfirmationData{
		Email: "test@example.com",
		Code:  "123456",
	}

	input := SendEmailConfirmationInput{
		To:      "test@example.com",
		Subject: "テスト件名",
		Body:    emails.EmailConfirmation(templateData),
	}

	err := sender.SendEmailConfirmation(ctx, input)
	if err != nil {
		t.Fatalf("SendEmailConfirmation failed: %v", err)
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
	ctx := templates.WithLocale(context.Background(), "ja")

	// 複数回メール送信
	for i := 0; i < 3; i++ {
		templateData := emails.EmailConfirmationData{
			Email: "test@example.com",
			Code:  "123456",
		}
		input := SendEmailConfirmationInput{
			To:      "test@example.com",
			Subject: "テスト",
			Body:    emails.EmailConfirmation(templateData),
		}
		err := sender.SendEmailConfirmation(ctx, input)
		if err != nil {
			t.Fatalf("SendEmailConfirmation failed: %v", err)
		}
	}

	if len(sender.SentEmails) != 3 {
		t.Errorf("expected 3 sent emails, got %d", len(sender.SentEmails))
	}
}

func TestEmailConfirmationTemplate_Japanese(t *testing.T) {
	t.Parallel()

	// 日本語ロケールを設定（i18nはinit()で自動初期化される）
	ctx := templates.WithLocale(context.Background(), "ja")

	templateData := emails.EmailConfirmationData{
		Email: "user@example.com",
		Code:  "654321",
	}

	// テンプレートをレンダリング
	var buf bytes.Buffer
	err := emails.EmailConfirmation(templateData).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("EmailConfirmation render failed: %v", err)
	}

	html := buf.String()

	// 確認コードが含まれているか
	if !strings.Contains(html, "654321") {
		t.Error("expected confirmation code in HTML")
	}

	// HTMLタグが含まれているか（templは小文字に変換する）
	if !strings.Contains(html, "<!doctype html>") {
		t.Error("expected doctype in HTML")
	}

	// lang属性が日本語になっているか
	if !strings.Contains(html, `lang="ja"`) {
		t.Error("expected lang=ja in HTML")
	}
}

func TestEmailConfirmationTemplate_English(t *testing.T) {
	t.Parallel()

	// 英語ロケールを設定（i18nはinit()で自動初期化される）
	ctx := templates.WithLocale(context.Background(), "en")

	templateData := emails.EmailConfirmationData{
		Email: "user@example.com",
		Code:  "987654",
	}

	// テンプレートをレンダリング
	var buf bytes.Buffer
	err := emails.EmailConfirmation(templateData).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("EmailConfirmation render failed: %v", err)
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
}
