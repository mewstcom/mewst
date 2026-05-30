package post

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Create handles the new post submission (POST /posts). On success it sets a
// flash message and redirects to /home; on a validation error it re-renders the
// form with status 422; any other error becomes a 500.
//
// [Ja] Create は新規投稿の送信を処理する (POST /posts)。成功時は flash を設定して
// /home にリダイレクトし、バリデーションエラー時は 422 でフォームを再描画する。
// それ以外のエラーは 500 とする。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	canonicalURL := r.FormValue("canonical_url")

	// The author is the current viewer's profile, supplied by RequireAuth.
	// A missing profile means the auth middleware did not run as expected, so
	// treat it as an internal error rather than attributing the post to no one.
	//
	// [Ja] 投稿者は RequireAuth が供給する現在閲覧者のプロフィール。プロフィールが
	// 無いのは認証ミドルウェアが想定どおり動いていないことを意味するため、投稿者を
	// 不在のまま作成せず内部エラーとして扱う。
	profile := middleware.ProfileFromContext(ctx)
	if profile == nil {
		slog.ErrorContext(ctx, "現在プロフィールが context に存在しない")
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
			h.renderNewForm(w, r, ve, content, canonicalURL)
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
