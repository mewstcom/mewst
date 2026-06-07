package post

import (
	"net/http"
)

// New renders the new post form (GET /new).
//
// [Ja] New は新規投稿フォームを表示する (GET /new)。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	h.renderNewForm(w, r, nil, "", "", nil)
}
