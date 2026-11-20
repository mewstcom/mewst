package export_download

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mewstcom/mewst/go/internal/httperror"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// archiveContentTypeはエクスポートのアーカイブのメディアタイプ。明示的に
// 設定することで、サーバー自身が生成したのではないストリームの先頭バイトを
// net/httpが判定した結果にレスポンスが依存しないようにする。
const archiveContentType = "application/zip"

// archiveCacheControlはアーカイブをあらゆるキャッシュから外す。ルート
// グループはページを "private, no-cache" とし、ブラウザに複製の保持と再検証を
// 許すが、アーカイブはさらに踏み込む。読み手が一度ダウンロードするファイルであり、
// 新しいエクスポートが同じ場所でそれを置き換えるため、保存された複製は既に存在
// しないアーカイブとして返されることになるからである。
const archiveCacheControl = "private, no-store"

// Showはプロフィールのダウンロード可能なエクスポートのアーカイブを渡す
// (GET /settings/export/download)。
//
// リクエストの内容はログイン中のユーザーとプロフィールがすべてである。どの
// アーカイブが存在し、この読み手がそれを得てよいかはUseCaseがその組から判断する
// ため、このハンドラーはIDを比較することも、オブジェクトを自ら名指しすることも
// しない。ここで加えるのはレスポンスの説明 (メディアタイプ、ファイルアプリが表示
// するファイル名、ダウンロードされるファイルに必要なキャッシュとsniffingの規則)
// とストリームの転送である。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// RequireAuthがこのハンドラーの前に両方をcontextへ格納するため、欠けて
	// いる場合は未ログインではなくRequireAuth無しでルートを登録したことを意味する。
	// 所有者が不明なままアーカイブを開かず、500で失敗させる。
	user := middleware.UserFromContext(ctx)
	profile := middleware.ProfileFromContext(ctx)
	if user == nil || profile == nil {
		slog.ErrorContext(ctx, "エクスポートのダウンロードでログイン中のユーザーまたはプロフィールを取得できませんでした")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	output, err := h.getExportDownloadUC.Execute(ctx, usecase.GetExportDownloadInput{
		UserID:    user.ID,
		ProfileID: profile.ID,
	})
	if err != nil {
		handleShowError(w, r, err)
		return
	}
	defer func() {
		if err := output.Body.Close(); err != nil {
			slog.WarnContext(ctx, "エクスポートのストリームのクローズに失敗", "error", err)
		}
	}()

	// ファイル名はUseCaseが固定の書式と日付から組み立てるため、RFC 6266の
	// 拡張エンコーディングを要するものを含まない。クライアントへ1つのトークンとして
	// 渡すには引用符で囲めば足りる。
	w.Header().Set("Content-Type", archiveContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", output.FileName))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", archiveCacheControl)
	// サイズは最初のバイトを送る前に判明しているため、chunked転送に委ねず
	// 宣言する。長さが分かるクライアントは、時間のかかるダウンロードの進捗を表示でき、
	// 完全なアーカイブと途中で終わった接続を区別できる。
	w.Header().Set("Content-Length", strconv.FormatInt(output.Size, 10))
	w.WriteHeader(http.StatusOK)

	// バイト列はそのまま転送する。アーカイブは既に圧縮されており、レスポンスは
	// 長さを宣言しているため、ここで再エンコードするものは無い。
	//
	// 転送の失敗は、レスポンスボディを送り切れなかったこと (読み手がダウンロードを
	// 中断した、接続が切れた) か、オブジェクトのストリームが途中で終わったことを
	// 意味する。どちらもヘッダー送出後にサーバーが別の応答を返せるものではなく、
	// 前者はダウンロードの終わり方として通常のものであるためwarnで記録する。
	// Sentryはerror以上を送るため、これはアラートではなくローカルのログに残る。
	if _, err := io.Copy(w, output.Body); err != nil {
		slog.WarnContext(ctx, "エクスポートのアーカイブの転送に失敗", "error", err)
	}
}

// handleShowErrorは拒否または失敗したダウンロードをレスポンスに変換する。
//
// この時点でwには何も書き込んでいないため、どの結果もまだ自身のstatusと本文を
// 選べる。所有していないプロフィールと、アーカイブを持たないプロフィールは、
// どちらもUseCaseがnot foundとして答える。これが応答から両者を区別できないように
// している。
func handleShowError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	var ae *model.AppError
	if errors.As(err, &ae) {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound:
			httperror.NotFound(w, r)
		case model.AppErrCodeServiceUnavailable:
			// アーカイブを提供できないデプロイではダウンロードリンクを描画
			// しないため、ここへ到達したリクエストは現在の画面から来たものではない。
			// 機能を説明するページではなく、提供していないことを表すstatusで答える。
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "エクスポートのダウンロードに失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
