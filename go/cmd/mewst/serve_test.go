package main

import (
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/email"
)

// TestNewEmailSender_WithProviderConfigurationは配信する側の分岐を固定する。
// この関数はそのデプロイがメールを送るかどうかを決める唯一の箇所であり、ここでの
// 退行は静かに起きる。メールのジョブはすべて成功したまま、受信者には何も届かない。
func TestNewEmailSender_WithProviderConfiguration(t *testing.T) {
	t.Parallel()

	got := newEmailSender(&config.Config{
		ResendAPIKey: "dummy-api-key",
		EmailFrom:    "noreply@example.com",
	})
	if _, ok := got.(*email.ResendSender); !ok {
		t.Errorf("newEmailSender()の型 = %T、期待値 = *email.ResendSender", got)
	}
}

func TestNewEmailSender_WithoutProviderConfiguration(t *testing.T) {
	t.Parallel()

	tests := map[string]*config.Config{
		"missing API key": {
			EmailFrom: "noreply@example.com",
		},
		"missing sender address": {
			ResendAPIKey: "dummy-api-key",
		},
		"missing both": {},
	}

	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := newEmailSender(cfg)
			if _, ok := got.(*email.DiscardSender); !ok {
				t.Errorf("newEmailSender()の型 = %T、期待値 = *email.DiscardSender", got)
			}
		})
	}
}
