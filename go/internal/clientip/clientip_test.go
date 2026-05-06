package clientip_test

import (
	"net/http/httptest"
	"testing"

	"github.com/mewstcom/mewst/go/internal/clientip"
)

func TestGetClientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfIP       string
		xff        string
		xri        string
		remoteAddr string
		want       string
	}{
		{
			name:       "CF-Connecting-IPが設定されている場合",
			cfIP:       "203.0.113.1",
			xff:        "192.168.1.1",
			xri:        "10.0.0.1",
			remoteAddr: "127.0.0.1:8080",
			want:       "203.0.113.1",
		},
		{
			name:       "X-Forwarded-Forのみ設定されている場合（単一IP）",
			cfIP:       "",
			xff:        "192.168.1.1",
			xri:        "",
			remoteAddr: "127.0.0.1:8080",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Forwarded-Forのみ設定されている場合（複数IP）",
			cfIP:       "",
			xff:        "203.0.113.50, 192.168.1.1, 10.0.0.1",
			xri:        "",
			remoteAddr: "127.0.0.1:8080",
			want:       "203.0.113.50",
		},
		{
			name:       "X-Forwarded-Forのみ設定されている場合（複数IP、スペースなし）",
			cfIP:       "",
			xff:        "203.0.113.50,192.168.1.1",
			xri:        "",
			remoteAddr: "127.0.0.1:8080",
			want:       "203.0.113.50",
		},
		{
			name:       "X-Real-IPのみ設定されている場合",
			cfIP:       "",
			xff:        "",
			xri:        "10.0.0.1",
			remoteAddr: "127.0.0.1:8080",
			want:       "10.0.0.1",
		},
		{
			name:       "RemoteAddrのみ使用（ポート番号あり）",
			cfIP:       "",
			xff:        "",
			xri:        "",
			remoteAddr: "127.0.0.1:8080",
			want:       "127.0.0.1",
		},
		{
			name:       "RemoteAddrのみ使用（ポート番号なし）",
			cfIP:       "",
			xff:        "",
			xri:        "",
			remoteAddr: "127.0.0.1",
			want:       "127.0.0.1",
		},
		{
			name:       "IPv6アドレス（RemoteAddr）",
			cfIP:       "",
			xff:        "",
			xri:        "",
			remoteAddr: "[::1]:8080",
			want:       "[::1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "/", nil)
			if tt.cfIP != "" {
				req.Header.Set("CF-Connecting-IP", tt.cfIP)
			}
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			req.RemoteAddr = tt.remoteAddr

			got := clientip.GetClientIP(req)
			if got != tt.want {
				t.Errorf("GetClientIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
