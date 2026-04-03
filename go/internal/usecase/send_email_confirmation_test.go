package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mewstcom/mewst/go/internal/usecase"
)

// mockEmailConfirmationSender はテスト用のEmailConfirmationSender実装
type mockEmailConfirmationSender struct {
	calls []mockEmailConfirmationSendCall
	err   error
}

type mockEmailConfirmationSendCall struct {
	To     string
	Code   string
	Locale string
}

func (m *mockEmailConfirmationSender) Send(_ context.Context, to, code, locale string) error {
	m.calls = append(m.calls, mockEmailConfirmationSendCall{To: to, Code: code, Locale: locale})
	return m.err
}

func TestSendEmailConfirmationUsecase_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      usecase.SendEmailConfirmationInput
		senderErr  error
		wantErr    bool
		wantCalls  int
		wantTo     string
		wantCode   string
		wantLocale string
	}{
		{
			name: "正常系: 日本語ロケールでメール送信",
			input: usecase.SendEmailConfirmationInput{
				Email:  "test@example.com",
				Code:   "123456",
				Locale: "ja",
			},
			wantErr:    false,
			wantCalls:  1,
			wantTo:     "test@example.com",
			wantCode:   "123456",
			wantLocale: "ja",
		},
		{
			name: "正常系: 英語ロケールでメール送信",
			input: usecase.SendEmailConfirmationInput{
				Email:  "test@example.com",
				Code:   "654321",
				Locale: "en",
			},
			wantErr:    false,
			wantCalls:  1,
			wantTo:     "test@example.com",
			wantCode:   "654321",
			wantLocale: "en",
		},
		{
			name: "異常系: 空のメールアドレス",
			input: usecase.SendEmailConfirmationInput{
				Email:  "",
				Code:   "123456",
				Locale: "ja",
			},
			wantErr:   true,
			wantCalls: 0,
		},
		{
			name: "異常系: メール送信エラー",
			input: usecase.SendEmailConfirmationInput{
				Email:  "test@example.com",
				Code:   "123456",
				Locale: "ja",
			},
			senderErr: errors.New("送信失敗"),
			wantErr:   true,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sender := &mockEmailConfirmationSender{err: tt.senderErr}
			uc := usecase.NewSendEmailConfirmationUsecase(sender)

			err := uc.Execute(context.Background(), tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(sender.calls) != tt.wantCalls {
				t.Fatalf("sender.Send() called %d times, want %d", len(sender.calls), tt.wantCalls)
			}

			if tt.wantCalls > 0 && !tt.wantErr {
				call := sender.calls[0]
				if call.To != tt.wantTo {
					t.Errorf("To = %q, want %q", call.To, tt.wantTo)
				}
				if call.Code != tt.wantCode {
					t.Errorf("Code = %q, want %q", call.Code, tt.wantCode)
				}
				if call.Locale != tt.wantLocale {
					t.Errorf("Locale = %q, want %q", call.Locale, tt.wantLocale)
				}
			}
		})
	}
}
