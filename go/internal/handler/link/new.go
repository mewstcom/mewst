package link

import (
	"net/http"
)

// New renders the add-link-card prompt fragment (GET /links/new). The target
// URL detected in the post body arrives as the "url" query parameter, mirroring
// the Rails Links::NewController (params[:url]). No validation happens here;
// the URL is validated when the prompt is submitted to POST /links.
//
// [Ja] New はリンクカード追加プロンプトのフラグメントを表示する (GET /links/new)。
// 投稿本文から検出した対象 URL はクエリパラメータ "url" で渡される (Rails の
// Links::NewController の params[:url] に対応)。ここではバリデーションを行わず、
// プロンプトが POST /links に送信された時点で URL を検証する。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	h.renderNewFragment(w, r, nil, r.URL.Query().Get("url"))
}
