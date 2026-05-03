package email_confirmation

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/redirect"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	emailconfirmationpages "github.com/mewstcom/mewst/go/internal/templates/pages/email_confirmation"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Create は確認コードを検証する (POST /email_confirmation)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// クッキーからemail_confirmation_idを取得
	id, ok := h.sessionMgr.GetEmailConfirmationID(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// フォームデータを取得
	code := r.FormValue("code")
	backURL := r.FormValue("back")

	// UseCase を実行（バリデーション + 確認成功マーク）
	ucResult, err := h.verifyEmailConfirmationUC.Execute(ctx, usecase.VerifyEmailConfirmationInput{
		ID:   id,
		Code: code,
	})
	if err != nil {
		h.handleCreateError(w, r, err, code, backURL)
		return
	}

	// イベントに応じたリダイレクト先を決定（sign_up イベントのみ back を伝搬）
	redirectPath := getRedirectPath(ucResult.EmailConfirmation.Event, backURL)

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_email_confirmed"))

	http.Redirect(w, r, redirectPath, http.StatusFound)
}

// handleCreateError はメール確認処理のエラーを処理する
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, code, backURL string) {
	ctx := r.Context()

	var ve *model.ValidationError
	if errors.As(err, &ve) {
		h.renderEmailConfirmationForm(w, r, ve, code, backURL)
		return
	}

	slog.ErrorContext(ctx, "メール確認処理に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// getRedirectPath はイベントに応じたリダイレクト先を返す。
//
// sign_up イベント時のみ backURL を伝搬する理由: 新規登録フローでは登録完了後にユーザーが元ページに戻ることを想定する。
// 一方、パスワードリセット・メール変更の完了画面は固定の遷移先（/sign_in もしくは /settings/email）であり、
// 元ページへ戻すこと自体を想定していないため backURL を伝搬しない。
func getRedirectPath(event model.EmailConfirmationEvent, backURL string) string {
	switch event {
	case model.EmailConfirmationEventPasswordReset:
		return "/password/edit"
	case model.EmailConfirmationEventSignUp:
		return redirect.AppendSafeBack("/accounts/new", backURL)
	case model.EmailConfirmationEventEmailUpdate:
		return "/settings/email"
	default:
		return "/"
	}
}

// renderEmailConfirmationForm は確認コード入力フォームを再表示する
func (h *Handler) renderEmailConfirmationForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, code, backURL string) {
	ctx := r.Context()
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	data := emailconfirmationpages.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: ve,
		Code:       code,
		BackURL:    backURL,
	}

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "email_confirmation_title")

	content := emailconfirmationpages.New(data)
	layout := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content)

	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := layout.Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
