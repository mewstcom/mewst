package post

import (
	"net/http"
)

// New renders the new post form (GET /new). It pre-fills the content textarea
// from the ?content= query parameter. Prefills are intentionally not validated
// on GET; validation occurs when the form is submitted to POST /posts.
//
// [Ja] New は新規投稿フォームを表示する (GET /new)。?content= クエリパラメータで
// 本文 textarea を事前入力する。GET では意図的に検証せず、POST /posts への送信時に
// 検証する。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	content := r.URL.Query().Get("content")
	h.renderNewForm(w, r, nil, content, "", nil)
}
