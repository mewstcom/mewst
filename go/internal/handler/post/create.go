package post

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// normalizeNewlinesはCRLFと単独のCRをLFに畳み、改行を1コードポイント
// にする。送信本文にこれが必要な理由は呼び出し箇所のコメントを参照。
func normalizeNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// Createは新規投稿の送信を処理する (POST /posts)。成功時はflashを設定して
// /homeにリダイレクトし、バリデーションエラー時は422でフォームを再描画する。
// それ以外のエラーは500とする。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 送信された改行をLFに正規化する。HTML仕様によりtextareaの改行は送信時
	// にCRLFへ正規化されるため、生の値を数える/検証する/保存すると改行1個が
	// 2コードポイント扱いになる。ここでCRLF (および単独のCR) をLFに畳むことで、
	// カウンター・文字数バリデーション・保存本文のすべてで改行が1コードポイントになる。
	content := normalizeNewlines(r.FormValue("content"))
	canonicalURL := r.FormValue("canonical_url")

	// 投稿者はRequireAuthが供給する現在閲覧者のプロフィール。プロフィールが
	// 無いのは認証ミドルウェアが想定どおり動いていないことを意味するため、投稿者を
	// 不在のまま作成せず内部エラーとして扱う。
	profile := middleware.ProfileFromContext(ctx)
	if profile == nil {
		slog.ErrorContext(ctx, "現在プロフィールがcontextに存在しない")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err := h.createPostUC.Execute(ctx, usecase.CreatePostInput{
		AuthorProfileID: profile.ID,
		Content:         content,
		CanonicalURL:    canonicalURL,
	})
	if err != nil {
		var ve *model.ValidationError
		if errors.As(err, &ve) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			h.renderNewForm(w, r, ve, content, canonicalURL, h.lookupAttachedLink(ctx, canonicalURL))
			return
		}
		var ae *model.AppError
		if errors.As(err, &ae) {
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		slog.ErrorContext(ctx, "投稿の作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_post_created"))
	http.Redirect(w, r, "/home", http.StatusFound)
}
