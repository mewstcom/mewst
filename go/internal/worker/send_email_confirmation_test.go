package worker

import (
	"context"
	"testing"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/email"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestSendEmailConfirmationWorker_Work(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    dispatcher.SendEmailConfirmationArgs
		wantErr bool
	}{
		{
			name: "正常系: 日本語ロケールでメール送信",
			args: dispatcher.SendEmailConfirmationArgs{
				Email:  "test@example.com",
				Code:   "123456",
				Locale: "ja",
			},
			wantErr: false,
		},
		{
			name: "正常系: 英語ロケールでメール送信",
			args: dispatcher.SendEmailConfirmationArgs{
				Email:  "test@example.com",
				Code:   "123456",
				Locale: "en",
			},
			wantErr: false,
		},
		{
			name: "正常系: 未知のロケールは日本語にフォールバック",
			args: dispatcher.SendEmailConfirmationArgs{
				Email:  "test@example.com",
				Code:   "123456",
				Locale: "unknown",
			},
			wantErr: false,
		},
		{
			name: "異常系: 空のメールアドレス",
			args: dispatcher.SendEmailConfirmationArgs{
				Email:  "",
				Code:   "123456",
				Locale: "ja",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			noopSender := email.NewNoopSender()
			confirmationSender := email.NewConfirmationSender(noopSender)
			uc := usecase.NewSendEmailConfirmationUsecase(confirmationSender)
			w := NewSendEmailConfirmationWorker(uc)

			job := &river.Job[dispatcher.SendEmailConfirmationArgs]{
				Args: tt.args,
			}

			err := w.Work(context.Background(), job)

			if (err != nil) != tt.wantErr {
				t.Errorf("Work() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// メールが送信されたことを確認
				if len(noopSender.SentEmails) != 1 {
					t.Errorf("SentEmails count = %v, want 1", len(noopSender.SentEmails))
					return
				}

				sentEmail := noopSender.SentEmails[0]
				if sentEmail.To != tt.args.Email {
					t.Errorf("SentEmail.To = %v, want %v", sentEmail.To, tt.args.Email)
				}

				// 件名を検証
				expectedSubject := "[Mewst] 確認用コード"
				if tt.args.Locale == "en" {
					expectedSubject = "[Mewst] Confirmation code"
				}
				if sentEmail.Subject != expectedSubject {
					t.Errorf("SentEmail.Subject = %v, want %v", sentEmail.Subject, expectedSubject)
				}
			}
		})
	}
}
