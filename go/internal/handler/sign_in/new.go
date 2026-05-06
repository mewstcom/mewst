package sign_in

import (
	"net/http"
)

// New はログインフォームを表示する (GET /sign_in)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	backURL := r.URL.Query().Get("back")
	h.renderSignInForm(w, r, nil, "", backURL)
}
