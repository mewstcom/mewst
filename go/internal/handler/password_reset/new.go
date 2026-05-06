package password_reset

import (
	"net/http"
)

// New はパスワードリセットフォームを表示する (GET /password_reset)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	h.renderPasswordResetForm(w, r, nil, "")
}
