package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/database"
	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/email"
	"github.com/mewstcom/mewst/go/internal/exportfile"
	"github.com/mewstcom/mewst/go/internal/handler/account"
	"github.com/mewstcom/mewst/go/internal/handler/email_confirmation"
	"github.com/mewstcom/mewst/go/internal/handler/export"
	"github.com/mewstcom/mewst/go/internal/handler/link"
	"github.com/mewstcom/mewst/go/internal/handler/manifest"
	"github.com/mewstcom/mewst/go/internal/handler/password"
	"github.com/mewstcom/mewst/go/internal/handler/password_reset"
	"github.com/mewstcom/mewst/go/internal/handler/post"
	"github.com/mewstcom/mewst/go/internal/handler/setting"
	"github.com/mewstcom/mewst/go/internal/handler/sign_in"
	"github.com/mewstcom/mewst/go/internal/handler/sign_out"
	"github.com/mewstcom/mewst/go/internal/handler/sign_up"
	"github.com/mewstcom/mewst/go/internal/httperror"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/repository"
	mewstsentry "github.com/mewstcom/mewst/go/internal/sentry"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/storage"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
	"github.com/mewstcom/mewst/go/internal/worker"
)

func main() {
	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		os.Exit(1)
	}

	slog.Info("サーバーを起動します", "port", cfg.Port, "env", cfg.Env)

	// Sentry を初期化 (DSN が空の場合はスキップされる)。
	// データベース接続より前に初期化することで、起動時のエラーも Sentry に送信できる。
	if err := mewstsentry.Init(mewstsentry.Config{
		DSN:              cfg.SentryDSN,
		Environment:      cfg.SentryEnvironment,
		Release:          cfg.AssetVersion,
		TracesSampleRate: cfg.SentryTracesSampleRate,
		Debug:            cfg.SentryDebug,
	}); err != nil {
		slog.Error("Sentry の初期化に失敗しました", "error", err)
		os.Exit(1)
	}
	defer mewstsentry.Flush(2 * time.Second)

	// slog のデフォルトハンドラーを「標準出力 + Sentry」のファンアウトに差し替える。
	// これ以降の slog.ErrorContext / slog.Error 呼び出しは自動的に Sentry のイベントとして送信される。
	// 各層 (handler / usecase / validator / repository / middleware) で sentry.CaptureError を
	// 明示的に呼ぶ必要がない。
	// Sentry 初期化失敗時のログは引き続き Go 1.21+ のデフォルトハンドラー (TextHandler @ stderr) に出すべきため、
	// SetDefault は mewstsentry.Init 成功後に呼ぶ。
	baseSlogHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(mewstsentry.NewSlogHandler(baseSlogHandler)))

	// データベース接続
	db, err := database.Connect(cfg.DatabaseDSN())
	if err != nil {
		slog.Error("データベース接続に失敗しました", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("データベース接続のクローズに失敗しました", "error", err)
		}
	}()
	slog.Info("データベースに接続しました")

	// クエリの初期化
	queries := query.New(db)

	// リポジトリの初期化
	userRepo := repository.NewUserRepository(queries)
	sessionRepo := repository.NewSessionRepository(queries)
	actorRepo := repository.NewActorRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	profileRepo := repository.NewProfileRepository(queries)
	userProfileRepo := repository.NewUserProfileRepository(queries)
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	featureFlagRepo := repository.NewFeatureFlagRepository(queries)
	postRepo := repository.NewPostRepository(queries)
	followRepo := repository.NewFollowRepository(queries)
	homeTimelinePostRepo := repository.NewHomeTimelinePostRepository(queries)
	oauthApplicationRepo := repository.NewOauthApplicationRepository(queries)
	linkRepo := repository.NewLinkRepository(queries)
	postLinkRepo := repository.NewPostLinkRepository(queries)
	exportRepo := repository.NewExportRepository(queries)
	exportCompletionNotificationRepo := repository.NewExportCompletionNotificationRepository(queries)
	exportProfileDeletionGuardRepo := repository.NewExportProfileDeletionGuardRepository(db)
	exportPostRepo := repository.NewExportPostRepository(queries)

	// セッションマネージャーの初期化
	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// Build the Dispatcher first, backed by a DeferredInserter. FanoutPostUsecase
	// needs the Dispatcher, the Dispatcher needs the River client, and the River
	// client can only be built after worker.NewClient registers the Workers that
	// wrap those UseCases — an initialization cycle. Construct the Dispatcher
	// around an unwired DeferredInserter now and inject the River client via
	// SetInserter once worker.NewClient has created it.
	//
	// [Ja] FanoutPostUsecase → Dispatcher → River クライアント → Worker (UseCase を内包) の
	// 初期化循環を断つため、先に DeferredInserter で Dispatcher を構築し、River クライアント
	// 生成後に注入する。
	deferredInserter := &dispatcher.DeferredInserter{}
	jobDispatcher := dispatcher.NewDispatcher(deferredInserter)

	// Build the fanout UseCases that the Workers register. They depend on
	// repository, so they cannot be built inside the worker package (depguard
	// forbids worker → repository); build them here and inject them.
	// [Ja] Worker に登録する fanout 系 UseCase を構築する (repository 依存のため worker 内で
	// は構築できず、ここで注入する)。
	fanoutPostUC := usecase.NewFanoutPostUsecase(postRepo, followRepo, jobDispatcher)
	addPostToTimelineUC := usecase.NewAddPostToTimelineUsecase(profileRepo, postRepo, homeTimelinePostRepo)

	// The mail sender is built here rather than inside worker.NewClient because
	// the export completion mail is sent by a UseCase that reads the notification
	// outbox, which puts it among the repository-dependent UseCases built above.
	//
	// [Ja] メール sender を worker.NewClient 内ではなくここで構築する。エクスポート
	// 完了メールを送るのは通知 outbox を読む UseCase であり、上で構築している
	// repository 依存の UseCase 群に属するため。
	emailSender := newEmailSender(cfg)
	// Resolve once whether this deployment can run exports, and give the same
	// value to everything that depends on it: the Worker registration below and
	// the export page further down. One expression means the page cannot offer
	// an action whose Worker was never registered.
	//
	// [Ja] このデプロイがエクスポートを実行できるかを一度だけ解決し、それに依存する
	// すべて (下の Worker 登録と、後ろのエクスポート画面) へ同じ値を渡す。式が 1 つ
	// であれば、Worker が登録されていない操作を画面が出すことはあり得ない。
	exportStorageReady := cfg.S3Readiness() == config.S3ReadinessReady
	exportUCs := newExportUsecases(
		cfg,
		exportStorageReady,
		emailSender,
		exportRepo,
		exportCompletionNotificationRepo,
		exportProfileDeletionGuardRepo,
		exportPostRepo,
		actorRepo,
		userRepo,
		jobDispatcher,
	)

	// Workerの初期化
	workerClient, err := worker.NewClient(context.Background(), cfg.DatabaseDSN(), emailSender, fanoutPostUC, addPostToTimelineUC, exportUCs)
	if err != nil {
		slog.Error("Workerクライアントの初期化に失敗しました", "error", err)
		os.Exit(1)
	}

	// Wire the real inserter into the DeferredInserter now that the River client
	// exists (completes the wiring that broke the init cycle).
	// [Ja] River クライアント生成後に DeferredInserter へ実体を注入する (循環を断った配線の完了)。
	deferredInserter.SetInserter(workerClient.Client())

	// Workerを開始
	if err := workerClient.Start(context.Background()); err != nil {
		slog.Error("Workerの開始に失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("Workerを開始しました")

	// レートリミッターの初期化
	rateLimiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターの初期化
	signInValidator := validator.NewSignInCreateValidator(userRepo)
	signUpValidator := validator.NewSignUpCreateValidator(userRepo)
	emailConfirmationValidator := validator.NewEmailConfirmationCreateValidator(emailConfirmationRepo)
	accountValidator := validator.NewAccountCreateValidator(userRepo, profileRepo)
	passwordUpdateValidator := validator.NewPasswordUpdateValidator()
	passwordResetCreateValidator := validator.NewPasswordResetCreateValidator()
	postCreateValidator := validator.NewPostCreateValidator()
	linkDataFetcherValidator := validator.NewLinkDataFetcherValidator()

	// ユースケースの初期化
	createSessionUC := usecase.NewCreateSessionUsecase(actorRepo, sessionRepo)
	deleteSessionUC := usecase.NewDeleteSessionUsecase(sessionRepo)
	createSignInUC := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)
	createSignUpUC := usecase.NewCreateSignUpUsecase(signUpValidator, emailConfirmationRepo, jobDispatcher)
	createPasswordResetUC := usecase.NewCreatePasswordResetUsecase(passwordResetCreateValidator, emailConfirmationRepo, jobDispatcher)
	verifyEmailConfirmationUC := usecase.NewVerifyEmailConfirmationUsecase(emailConfirmationValidator, emailConfirmationRepo)
	getActiveEmailConfirmationUC := usecase.NewGetActiveEmailConfirmationUsecase(emailConfirmationRepo)
	getSucceededEmailConfirmationUC := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmationRepo)
	updatePasswordUC := usecase.NewUpdatePasswordUsecase(passwordUpdateValidator, userRepo)
	createAccountUC := usecase.NewCreateAccountUsecase(db, accountValidator, userRepo, profileRepo, userProfileRepo, actorRepo)
	createPostUC := usecase.NewCreatePostUsecase(db, postCreateValidator, oauthApplicationRepo, linkRepo, postRepo, postLinkRepo, profileRepo, homeTimelinePostRepo, jobDispatcher)
	getLinkUC := usecase.NewGetLinkUsecase(linkRepo)
	// blockPrivateHosts is true in production wiring: fetching user-supplied URLs
	// must not reach internal hosts (SSRF). Passing false still compiles and
	// passes tests, so review wiring changes here carefully. The 10s timeout
	// bounds each fetch (applied per redirect hop) so a slow external site cannot
	// pin a request handler indefinitely.
	//
	// [Ja] 本番配線では blockPrivateHosts を true にする。ユーザー入力の URL の
	// 取得が内部ホストへ到達してはならない (SSRF 対策)。false を渡しても
	// コンパイル・テストは通るため、この配線の変更は注意して見ること。10 秒の
	// タイムアウトは取得 1 回ごと (リダイレクトの各ホップ) に適用され、遅い外部
	// サイトがリクエストハンドラーを占有し続けないようにする。
	fetchLinkMetadataUC := usecase.NewFetchLinkMetadataUsecase(linkDataFetcherValidator, linkRepo, &http.Client{Timeout: 10 * time.Second}, true)
	// The export page is given the same readiness the export Workers are gated
	// on, so a deployment without MEWST_S3_* tells the reader the feature is
	// unavailable instead of offering actions that cannot complete.
	//
	// [Ja] エクスポート画面にはエクスポート系 Worker の登録と同じ readiness を渡す。
	// MEWST_S3_* が無いデプロイでは、完了し得ない操作を出す代わりに、機能が利用
	// できないことを読み手へ伝える。
	getExportShowUC := usecase.NewGetExportShowUsecase(userProfileRepo, exportRepo, exportStorageReady)
	// The settings menu reads the export feature flag so its export entry
	// appears only for the actors the reverse proxy also routes to the Go
	// export page.
	//
	// [Ja] 設定メニューはエクスポートのフィーチャーフラグを読み、リバースプロキシが
	// Go 版のエクスポート画面へ振り分ける actor にだけエクスポート項目を出す。
	getSettingIndexUC := usecase.NewGetSettingIndexUsecase(featureFlagRepo)
	// Starting an export is gated on the same readiness, so a deployment
	// without MEWST_S3_* refuses the request instead of persisting a queued
	// export no Worker is registered to generate.
	//
	// [Ja] エクスポートの開始も同じ readiness でゲートする。MEWST_S3_* が無い
	// デプロイでは、生成する Worker が登録されていない queued のエクスポートを
	// 永続化する代わりに、リクエストを拒否する。
	createExportUC := usecase.NewCreateExportUsecase(db, userProfileRepo, exportRepo, jobDispatcher, exportStorageReady)

	// Turnstileクライアントの初期化
	turnstileClient := turnstile.NewClient(cfg.TurnstileSecretKey)

	// ハンドラーの初期化
	manifestHandler := manifest.NewHandler(cfg)
	signInHandler := sign_in.NewHandler(cfg, sessionMgr, flashMgr, createSignInUC, turnstileClient)
	signUpHandler := sign_up.NewHandler(cfg, sessionMgr, flashMgr, createSignUpUC, turnstileClient, rateLimiter)
	signOutHandler := sign_out.NewHandler(sessionMgr, flashMgr, deleteSessionUC)
	passwordResetHandler := password_reset.NewHandler(cfg, sessionMgr, flashMgr, createPasswordResetUC, turnstileClient)
	emailConfirmationHandler := email_confirmation.NewHandler(cfg, sessionMgr, flashMgr, getActiveEmailConfirmationUC, verifyEmailConfirmationUC)
	passwordHandler := password.NewHandler(cfg, sessionMgr, flashMgr, getSucceededEmailConfirmationUC, updatePasswordUC)
	accountHandler := account.NewHandler(cfg, sessionMgr, flashMgr, getSucceededEmailConfirmationUC, createAccountUC, createSessionUC, turnstileClient, rateLimiter)
	postHandler := post.NewHandler(cfg, flashMgr, createPostUC, getLinkUC)
	linkHandler := link.NewHandler(fetchLinkMetadataUC, rateLimiter)
	settingHandler := setting.NewHandler(cfg, getSettingIndexUC)
	exportHandler := export.NewHandler(cfg, flashMgr, getExportShowUC, createExportUC)

	// ミドルウェアの初期化
	authMiddleware := middleware.NewAuth(sessionMgr)
	csrfMiddleware := middleware.NewCSRF(cfg)
	sentryUserContextMW := middleware.NewSentryUserContext(profileRepo)

	// Sentry の HTTP ミドルウェア。
	// Repanic: true により、panic を Sentry に送ったあと再 panic させて後続の Recoverer に処理を委ねる。
	sentryHTTPHandler := sentryhttp.New(sentryhttp.Options{Repanic: true})

	// Chiルーターの設定
	r := chi.NewRouter()

	// 基本ミドルウェア
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.RequestID)
	// Client IP is resolved on demand via internal/clientip (CF-Connecting-IP first),
	// so chi's IP middleware is intentionally not registered. chi's RealIP is also
	// deprecated for IP spoofing (GHSA-3fxj-6jh8-hvhx) and must not be reintroduced.
	//
	// [Ja] クライアント IP は internal/clientip (CF-Connecting-IP 優先) で都度解決するため、
	// chi の IP ミドルウェアは意図的に登録しない。chi の RealIP は IP spoofing
	// (GHSA-3fxj-6jh8-hvhx) のため deprecated でもあり、再導入しないこと。

	// Recoverer を outer (chi の Use では最初に登録 = チェーンの一番外側) に置く。
	// chi の Recoverer は http.ErrAbortHandler 以外を recover して 500 を書き、再 panic しないため、
	// sentryhttp より「外側」(= panic 伝搬の最後に届く位置) に置かないと、sentryhttp が panic を見られない。
	r.Use(chimiddleware.Recoverer)
	// sentryhttp は Recoverer より「あとに登録」= chi のチェーンでは innermost (= handler に近い側)。
	// handler の panic を sentryhttp の defer がまず捕捉し、Sentry に送信したあと Repanic: true で再 panic。
	// 再 panic は outer の Recoverer に到達し、Recoverer が 500 レスポンスを書いて終了する。
	r.Use(sentryHTTPHandler.Handle)
	// chi のルートパターン (例: "/users/{id}") を Sentry のトランザクション名に上書きする。
	// sentryhttp より「あとに登録」することで、defer (LIFO) のタイミングで sentryhttp の
	// transaction.Finish() より先に Name を確定できる。
	r.Use(middleware.SentryTransaction)

	// リバースプロキシの設定（Rails版へのプロキシ）
	if cfg.RailsAppURL != "" {
		proxyMiddleware, err := middleware.NewReverseProxyMiddleware(cfg.RailsAppURL, cfg, featureFlagRepo)
		if err != nil {
			slog.Error("リバースプロキシミドルウェアの初期化に失敗しました", "error", err)
			os.Exit(1)
		}
		r.Use(proxyMiddleware.Middleware)
	}

	// Limit the request body size for Go-handled routes. Placed after reverse_proxy so
	// requests proxied to the Rails version are not affected, and before any middleware /
	// handler that parses form data (CSRF, MethodOverride, handlers).
	//
	// [Ja] Go 版が処理するルートのリクエストボディサイズを制限する。reverse_proxy より
	// 後に配置することで Rails 版にプロキシされるリクエストには影響させず、フォームを
	// パースするミドルウェア / ハンドラー (CSRF、MethodOverride、各ハンドラー) より前に置く。
	r.Use(middleware.BodyLimit)

	// i18n ミドルウェア(Accept-Language ヘッダーから ctx にロケールをセットする)
	// reverse_proxy より後に配置することで、Rails 版にプロキシされるリクエストには走らせない
	r.Use(i18n.Middleware)

	// フラッシュメッセージをCookieからcontextへロード（Go版の全ルートに適用）
	r.Use(flashMgr.Middleware)

	// 404ハンドラーの設定（ルーティングにマッチしないパス用）
	r.NotFound(httperror.NotFound)

	// 静的ファイルの配信 (Tailwind CLI + esbuild のビルド結果)
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// ヘルスチェックエンドポイント
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// DBの接続確認
		if err := db.PingContext(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "ヘルスチェック: データベース接続エラー", "error", err)
			http.Error(w, "DB connection failed", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Web App Manifest
	r.Get("/manifest.json", manifestHandler.Show)

	// ログイン・サインアップページ（未認証ユーザーのみ）
	r.Group(func(r chi.Router) {
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireNoAuth)
		r.Use(sentryUserContextMW.Middleware)
		r.Get("/sign_in", signInHandler.New)
		r.Post("/sign_in", signInHandler.Create)
		r.Get("/sign_up", signUpHandler.New)
		r.Post("/sign_up", signUpHandler.Create)
	})

	// Sign out (authenticated users only). The sign-out form is served by the Go
	// /settings page and submits with method="POST", so the request reaches this
	// server as POST. DELETE is registered alongside POST for the same handler,
	// mirroring how /password registers both PATCH and a POST fallback for HTML
	// forms.
	//
	// The CSRF middleware protects this endpoint. The /settings page runs under
	// the same CSRF middleware, so the token cookie is issued when that page loads
	// and its sign-out form embeds the matching token.
	//
	// [Ja] ログアウト (認証済みユーザーのみ)。ログアウトフォームは Go 版の /settings
	// ページが供給し、method="POST" で送信するため、リクエストは POST で本サーバーに
	// 届く。DELETE も同じハンドラーに登録しており、これは /password が PATCH と HTML
	// フォーム用の POST を二重登録しているのと同じ扱いである。
	//
	// CSRF ミドルウェアがこのエンドポイントを保護する。/settings ページも同じ CSRF
	// ミドルウェア配下で動くため、ページ読み込み時にトークン Cookie が発行され、
	// ログアウトフォームには一致するトークンが埋め込まれる。
	r.Group(func(r chi.Router) {
		r.Use(middleware.PrivateCache)
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireAuth)
		r.Use(sentryUserContextMW.Middleware)
		r.Delete("/sign_out", signOutHandler.Delete)
		r.Post("/sign_out", signOutHandler.Delete)
	})

	// パスワードリセット・メール確認・アカウント作成（未認証ユーザーのみ）
	r.Group(func(r chi.Router) {
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireNoAuth)
		r.Use(sentryUserContextMW.Middleware)

		// パスワードリセット開始（メールアドレス入力）
		r.Get("/password_reset", passwordResetHandler.New)
		r.Post("/password_reset", passwordResetHandler.Create)

		// メール確認（確認コード入力）
		r.Get("/email_confirmation", emailConfirmationHandler.New)
		r.Post("/email_confirmation", emailConfirmationHandler.Create)

		// パスワード更新（新しいパスワード設定）
		r.Get("/password/edit", passwordHandler.Edit)
		r.Patch("/password", passwordHandler.Update)
		r.Post("/password", passwordHandler.Update) // HTMLフォームからのPOST対応（_method=PATCH）

		// アカウント作成（メール確認後）
		r.Get("/accounts/new", accountHandler.New)
		r.Post("/accounts", accountHandler.Create)
	})

	// New post form, submission, and link card fragments (authenticated users
	// only). The reverse proxy routes these paths to Go unconditionally via its
	// goHandledPatterns (exact path + method), so they are no longer behind a
	// feature flag. POST /posts and POST /links are genuine POSTs, so no Method
	// Override is needed; the routes are registered directly. GET /links/new sits
	// under the CSRF middleware so the fragment can embed the CSRF token for its
	// POST /links form.
	//
	// [Ja] 新規投稿フォーム・送信・リンクカードのフラグメント（認証済みユーザーのみ）。
	// リバースプロキシはこれらのパスを goHandledPatterns (完全一致 + メソッド) で
	// 無条件に Go 版へ振り分けるため、もうフィーチャーフラグの配下にはない。
	// POST /posts と POST /links は本来の POST のため Method Override は不要で、
	// ルートを直接登録する。GET /links/new はフラグメントが POST /links フォーム用の
	// CSRF トークンを埋め込めるよう CSRF ミドルウェア配下に置く。
	r.Group(func(r chi.Router) {
		r.Use(middleware.PrivateCache)
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireAuth)
		r.Use(sentryUserContextMW.Middleware)

		r.Get("/new", postHandler.New)
		r.Post("/posts", postHandler.Create)
		r.Get("/links/new", linkHandler.New)
		r.Post("/links", linkHandler.Create)
	})

	// Settings pages (authenticated users only). The CSRF middleware makes its
	// token available to the sign-out and export forms, while RequireAuth
	// provides the current user and profile used by the navbar and by the
	// export page's authorization.
	//
	// [Ja] 設定系ページ (認証済みユーザーのみ)。CSRF ミドルウェアはログアウトと
	// エクスポートのフォームへトークンを渡し、RequireAuth は navbar とエクスポート
	// 画面の認可が使う現在のユーザーとプロフィールを渡す。
	r.Group(func(r chi.Router) {
		r.Use(middleware.PrivateCache)
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireAuth)
		r.Use(sentryUserContextMW.Middleware)

		r.Get("/settings", settingHandler.Index)
		r.Get("/settings/export", exportHandler.Show)
		r.Post("/settings/export", exportHandler.Create)
	})

	// サーバー起動
	// Dockerコンテナ内で動かす場合、0.0.0.0でリッスンする必要がある
	addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	slog.Info("HTTPサーバーを起動します", "addr", addr)

	// HTTPサーバーの作成
	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Graceful shutdown のためのシグナルハンドリング
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		slog.Info("シャットダウンシグナルを受信しました。サーバーを停止します...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Workerを停止
		if err := workerClient.Stop(shutdownCtx); err != nil {
			slog.Error("Workerの停止に失敗しました", "error", err)
		} else {
			slog.Info("Workerを停止しました")
		}

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("サーバーのシャットダウンに失敗しました", "error", err)
		}
	}()

	// サーバー起動
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("サーバーの起動に失敗しました", "error", err)
		os.Exit(1)
	}

	slog.Info("サーバーが正常に停止しました")
}

// newEmailSender returns the sender that delivers the app's mail, or the one
// that discards it when the provider is not configured. Every mail kind goes
// through the same instance, so whether a deployment delivers mail at all is
// decided once.
//
// [Ja] newEmailSender はアプリのメールを配信する sender を返す。プロバイダーが未設定の
// 場合はメールを破棄する sender を返す。すべての種類のメールが同じインスタンスを通るため、
// そのデプロイがメールを配信するかどうかの判断は 1 度だけ行われる。
func newEmailSender(cfg *config.Config) email.Sender {
	if cfg.ResendAPIKey == "" || cfg.EmailFrom == "" {
		slog.Warn("Resend APIキーまたは送信元メールアドレスが設定されていないため、メール送信は無効です")
		return email.NewDiscardSender()
	}

	slog.Info("Resend クライアントを初期化しました")
	return email.NewResendSender(cfg.ResendAPIKey, cfg.EmailFrom, cfg.EmailFromName)
}

// newExportUsecases builds the export UseCases the Worker runs, or returns the
// zero value when storageReady is false. The caller resolves that flag from
// cfg.S3Readiness once, so the Workers registered here and the export page are
// gated on the same value rather than on two expressions that can drift apart.
// The zero value keeps the feature flag's off state deployable without any
// MEWST_S3_* value: no export Worker is registered, no export periodic job is
// scheduled, and no export work runs. A partial configuration never reaches
// here, because config.Load rejects it at startup.
//
// They are built together even though reconciliation is the one that never
// reaches the object storage. Without it no export row can be created at all,
// so reconciling exports would be recovering work that cannot exist.
//
// [Ja] newExportUsecases は Worker が実行するエクスポート系 UseCase を構築する。
// storageReady が false の場合はゼロ値を返す。このフラグは呼び出し側が
// cfg.S3Readiness から一度だけ解決するため、ここで登録する Worker とエクスポート
// 画面は、乖離しうる 2 つの式ではなく同じ値でゲートされる。ゼロ値を返すことで、
// MEWST_S3_* を 1 つも設定しないままフィーチャーフラグ OFF の状態をデプロイできる
// (エクスポート系 Worker も定期ジョブも登録されず、エクスポートの処理は動作しない)。
// 一部だけ設定された状態は config.Load が起動時に拒否するため、ここには到達しない。
//
// オブジェクトストレージに触れないのはリコンシリエーションだけだが、これらはまとめて
// 構築する。ストレージが無ければエクスポート行自体を作成できないため、
// リコンシリエーションは存在し得ない処理を回復することになるからである。
func newExportUsecases(
	cfg *config.Config,
	storageReady bool,
	emailSender email.Sender,
	exportRepo *repository.ExportRepository,
	exportCompletionNotificationRepo *repository.ExportCompletionNotificationRepository,
	exportProfileDeletionGuard usecase.ExportProfileDeletionGuard,
	exportPostRepo *repository.ExportPostRepository,
	actorRepo *repository.ActorRepository,
	userRepo *repository.UserRepository,
	jobDispatcher *dispatcher.Dispatcher,
) worker.ExportUsecases {
	if !storageReady {
		return worker.ExportUsecases{}
	}

	exportStorage := storage.NewS3ExportStorage(storage.S3Config{
		BucketName:      cfg.S3BucketName,
		Endpoint:        cfg.S3Endpoint,
		AccessKeyID:     cfg.S3AccessKeyID,
		SecretAccessKey: cfg.S3SecretAccessKey,
		Region:          cfg.S3Region,
	})

	return worker.ExportUsecases{
		Generate: usecase.NewGenerateExportUsecase(
			exportRepo,
			exportPostRepo,
			actorRepo,
			userRepo,
			exportProfileDeletionGuard,
			exportfile.NewBuilder(),
			exportStorage,
			jobDispatcher,
		),
		CleanupOld: usecase.NewCleanupOldExportsUsecase(
			exportRepo,
			exportStorage,
		),
		SendCompletedEmail: usecase.NewSendExportCompletedEmailUsecase(
			exportCompletionNotificationRepo,
			exportProfileDeletionGuard,
			email.NewExportCompletedSender(emailSender),
			cfg.AppURL()+string(templates.SettingExportPath()),
		),
		Reconcile: usecase.NewReconcileExportsUsecase(
			exportRepo,
			exportCompletionNotificationRepo,
			jobDispatcher,
			usecase.DefaultExportRecoveryLimits(),
		),
		CleanupOrphanObjects: usecase.NewCleanupOrphanExportObjectsUsecase(
			exportRepo,
			exportStorage,
			jobDispatcher,
			usecase.DefaultExportOrphanSweepLimits(),
		),
	}
}
