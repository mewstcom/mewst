package post

import (
	"net/http"
)

// Newは新規投稿フォームを表示する (GET /new)。?content= クエリパラメータで
// 本文textareaを事前入力する。GETでは意図的に検証せず、POST /postsへの送信時に
// 検証する。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	content := r.URL.Query().Get("content")
	h.renderNewForm(w, r, nil, content, "", nil)
}
