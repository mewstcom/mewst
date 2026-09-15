package link

import (
	"net/http"
)

// Newはリンクカード追加プロンプトのフラグメントを表示する (GET /links/new)。
// 投稿本文から検出した対象URLはクエリパラメータ "url" で渡される (Railsの
// Links::NewControllerのparams[:url] に対応)。ここではバリデーションを行わず、
// プロンプトがPOST /linksに送信された時点でURLを検証する。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	h.renderNewFragment(w, r, nil, r.URL.Query().Get("url"))
}
