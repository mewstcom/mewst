package export

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/httperror"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Create starts an export for the signed-in profile (POST /settings/export).
// The form carries no fields beyond the CSRF token the middleware verifies, so
// the whole request is the identity on the context: the user and profile decide
// whose posts are exported and whether that is allowed, and the actor is
// recorded as the requester.
//
// Every outcome ends at the export page, which is where the state this request
// produced is described. Success and refusals differ in the flash they leave
// behind, not in where the reader lands.
//
// [Ja] Create はログイン中プロフィールのエクスポートを開始する
// (POST /settings/export)。フォームはミドルウェアが検証する CSRF トークン以外の
// フィールドを持たないため、リクエストの内容は context 上の identity がすべてである。
// ユーザーとプロフィールが「誰のポストをエクスポートするか」と「それが許されるか」を
// 決め、actor は申請者として記録される。
//
// どの結果もエクスポート画面へ着く。このリクエストが生んだ状態を説明するのはその
// 画面であるため。成功と拒否で異なるのは残す flash であって、読み手の行き先ではない。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// RequireAuth puts all three on the context before this handler runs, so a
	// missing one means the route was wired without it rather than a signed-out
	// visitor. Fail with 500 instead of creating an export for an unknown owner.
	//
	// [Ja] RequireAuth がこのハンドラーの前に 3 つとも context へ格納するため、
	// 欠けている場合は未ログインではなく RequireAuth 無しでルートを登録したことを
	// 意味する。所有者が不明なままエクスポートを作らず、500 で失敗させる。
	user := middleware.UserFromContext(ctx)
	profile := middleware.ProfileFromContext(ctx)
	actor := middleware.ActorFromContext(ctx)
	if user == nil || profile == nil || actor == nil {
		slog.ErrorContext(ctx, "エクスポート開始でログイン中のユーザー・プロフィール・アクターを取得できませんでした")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := h.createExportUC.Execute(ctx, usecase.CreateExportInput{
		UserID:    user.ID,
		ProfileID: profile.ID,
		ActorID:   actor.ID,
	}); err != nil {
		h.handleCreateError(w, r, err)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_export_started"))
	http.Redirect(w, r, string(templates.SettingExportPath()), http.StatusFound)
}

// handleCreateError turns a failed export start into a response.
//
// A conflict is the profile's own state refusing the request (an export is
// already running, or the profile is being deleted), so the reader is sent back
// to the page that describes that state with the UseCase's message as a
// warning. The message comes from the error rather than from a key chosen here,
// because which conflict occurred is the UseCase's finding, and both conflicts
// leave the same trace in the HTTP response.
//
// [Ja] handleCreateError はエクスポート開始の失敗をレスポンスに変換する。
//
// 競合はプロフィール自身の状態による拒否 (既にエクスポートが実行中、あるいは
// プロフィールが削除処理中) であるため、その状態を説明する画面へ戻し、UseCase の
// メッセージを警告として見せる。メッセージをここで選んだキーではなくエラーから
// 取るのは、どの競合が起きたかが UseCase の判断であり、2 つの競合は HTTP
// レスポンスに同じ痕跡しか残さないためである。
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	var ae *model.AppError
	if errors.As(err, &ae) {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound:
			httperror.NotFound(w, r)
		case model.AppErrCodeConflict:
			slog.InfoContext(ctx, ae.LogString())
			h.flashMgr.SetWarning(w, ae.UserMsg)
			http.Redirect(w, r, string(templates.SettingExportPath()), http.StatusFound)
		case model.AppErrCodeServiceUnavailable:
			// The start button is not rendered on a deployment that cannot run
			// exports, so reaching here means the request did not come from the
			// page as it stands. Answer with the status that says the feature is
			// not being served rather than with a page describing it.
			//
			// [Ja] エクスポートを実行できないデプロイでは開始ボタンを描画しないため、
			// ここへ到達したリクエストは現在の画面から来たものではない。機能を説明する
			// ページではなく、提供していないことを表す status で答える。
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "エクスポートの開始に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
