package manifest

import (
	"encoding/json"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/i18n"
)

// Show はWeb App Manifestを返します (GET /manifest.json)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// アプリケーション名（開発環境では "(Dev)" を付ける）
	appName := "Mewst"
	if h.cfg.IsDev() {
		appName = "Mewst (Dev)"
	}

	// Web App Manifestを構築
	manifest := Manifest{
		BackgroundColor: "#f6f2eb",
		Description:     i18n.T(ctx, "default_description"),
		Display:         "standalone",
		Icons: []ManifestIcon{
			{
				Purpose: "any maskable",
				Sizes:   "192x192",
				Src:     "/static/images/icon-192.png",
				Type:    "image/png",
			},
			{
				Purpose: "any maskable",
				Sizes:   "512x512",
				Src:     "/static/images/icon-512.png",
				Type:    "image/png",
			},
		},
		Name:       appName,
		Scope:      "/",
		ShortName:  "Mewst",
		StartURL:   "/",
		ThemeColor: "#f6f2eb",
	}

	// JSONとしてレスポンスを返す
	w.Header().Set("Content-Type", "application/manifest+json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(manifest); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
