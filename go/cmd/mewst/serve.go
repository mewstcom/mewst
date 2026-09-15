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
	"github.com/mewstcom/mewst/go/internal/handler/export_download"
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

// runServeはHTTPサーバーを起動し、シャットダウンが完了するまでブロックする。
func runServe() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		os.Exit(1)
	}

	slog.Info("サーバーを起動します", "port", cfg.Port, "env", cfg.Env)

	// Sentryを初期化する (DSNが空の場合はスキップされる)。データベース接続より
	// 前に初期化することで、起動時のエラーもSentryに送信できる。
	if err := mewstsentry.Init(mewstsentry.Config{
		DSN:              cfg.SentryDSN,
		Environment:      cfg.SentryEnvironment,
		Release:          cfg.AssetVersion,
		TracesSampleRate: cfg.SentryTracesSampleRate,
		Debug:            cfg.SentryDebug,
	}); err != nil {
		slog.Error("Sentryの初期化に失敗しました", "error", err)
		os.Exit(1)
	}
	defer mewstsentry.Flush(2 * time.Second)

	// slogのデフォルトハンドラーを「標準エラー出力 + Sentry」のファンアウトに
	// 差し替える。これ以降のslog.ErrorContext / slog.Error呼び出しは自動的にSentryの
	// イベントとして送信されるため、各層 (handler / usecase / validator / repository /
	// middleware) でsentry.CaptureErrorを明示的に呼ぶ必要がない。
	//
	// SetDefaultをmewstsentry.Initの成功後に呼ぶのは、初期化に失敗したときのログを
	// 引き続きGo 1.21+ のデフォルトハンドラー (TextHandler @ stderr) へ出すため。
	baseSlogHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(mewstsentry.NewSlogHandler(baseSlogHandler)))

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

	queries := query.New(db)

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

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// FanoutPostUsecase → Dispatcher → Riverクライアント → Worker (UseCaseを内包) の
	// 初期化循環を断つため、先にDeferredInserterでDispatcherを構築し、Riverクライアント
	// 生成後に注入する。
	deferredInserter := &dispatcher.DeferredInserter{}
	jobDispatcher := dispatcher.NewDispatcher(deferredInserter)

	// Workerに登録するfanout系UseCaseを構築する (repository依存のためworker内で
	// は構築できず、ここで注入する)。
	fanoutPostUC := usecase.NewFanoutPostUsecase(postRepo, followRepo, jobDispatcher)
	addPostToTimelineUC := usecase.NewAddPostToTimelineUsecase(profileRepo, postRepo, homeTimelinePostRepo)

	// メールsenderをworker.NewClient内ではなくここで構築する。エクスポート
	// 完了メールを送るのは通知outboxを読むUseCaseであり、上で構築している
	// repository依存のUseCase群に属するため。
	emailSender := newEmailSender(cfg)
	// このデプロイがエクスポートを実行できるかを一度だけ解決し、それに依存する
	// すべて (下のWorker登録と、後ろのエクスポート画面) へ同じ値を渡す。式が1つ
	// であれば、Workerが登録されていない操作を画面が出すことはあり得ない。
	exportStorageReady := cfg.S3Readiness() == config.S3ReadinessReady
	exportStorage := newExportStorage(cfg, exportStorageReady)
	exportUCs := newExportUsecases(
		cfg,
		exportStorageReady,
		exportStorage,
		emailSender,
		exportRepo,
		exportCompletionNotificationRepo,
		exportProfileDeletionGuardRepo,
		exportPostRepo,
		actorRepo,
		userRepo,
		jobDispatcher,
	)

	workerClient, err := worker.NewClient(context.Background(), cfg.DatabaseDSN(), emailSender, fanoutPostUC, addPostToTimelineUC, exportUCs)
	if err != nil {
		slog.Error("Workerクライアントの初期化に失敗しました", "error", err)
		os.Exit(1)
	}

	// Riverクライアント生成後にDeferredInserterへ実体を注入する (循環を断った配線の完了)。
	deferredInserter.SetInserter(workerClient.Client())

	if err := workerClient.Start(context.Background()); err != nil {
		slog.Error("Workerの開始に失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("Workerを開始しました")

	rateLimiter := ratelimit.NewLimiter(rateLimitRepo)

	signInValidator := validator.NewSignInCreateValidator(userRepo)
	signUpValidator := validator.NewSignUpCreateValidator(userRepo)
	emailConfirmationValidator := validator.NewEmailConfirmationCreateValidator(emailConfirmationRepo)
	accountValidator := validator.NewAccountCreateValidator(userRepo, profileRepo)
	passwordUpdateValidator := validator.NewPasswordUpdateValidator()
	passwordResetCreateValidator := validator.NewPasswordResetCreateValidator()
	postCreateValidator := validator.NewPostCreateValidator()
	linkDataFetcherValidator := validator.NewLinkDataFetcherValidator()

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
	// 本番配線ではblockPrivateHostsをtrueにする。ユーザー入力のURLの
	// 取得が内部ホストへ到達してはならない (SSRF対策)。falseを渡しても
	// コンパイル・テストは通るため、この配線の変更は注意して見ること。10秒の
	// タイムアウトは取得1回ごと (リダイレクトの各ホップ) に適用され、遅い外部
	// サイトがリクエストハンドラーを占有し続けないようにする。
	fetchLinkMetadataUC := usecase.NewFetchLinkMetadataUsecase(linkDataFetcherValidator, linkRepo, &http.Client{Timeout: 10 * time.Second}, true)
	// エクスポート画面にはエクスポート系Workerの登録と同じreadinessを渡す。
	// MEWST_S3_* が無いデプロイでは、完了し得ない操作を出す代わりに、機能が利用
	// できないことを読み手へ伝える。
	getExportShowUC := usecase.NewGetExportShowUsecase(userProfileRepo, exportRepo, exportStorageReady)
	// エクスポートの開始も同じreadinessでゲートする。MEWST_S3_* が無い
	// デプロイでは、生成するWorkerが登録されていないqueuedのエクスポートを
	// 永続化する代わりに、リクエストを拒否する。
	createExportUC := usecase.NewCreateExportUsecase(db, userProfileRepo, exportRepo, jobDispatcher, exportStorageReady)
	// ダウンロードも画面・開始と同じreadinessでゲートする。MEWST_S3_* が無い
	// デプロイでは、持っていないオブジェクトストレージへ到達する代わりに、リクエストを
	// 拒否する。
	getExportDownloadUC := usecase.NewGetExportDownloadUsecase(userProfileRepo, userRepo, exportRepo, exportStorage, exportStorageReady)

	turnstileClient := turnstile.NewClient(cfg.TurnstileSecretKey)

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
	settingHandler := setting.NewHandler(cfg)
	exportHandler := export.NewHandler(cfg, flashMgr, getExportShowUC, createExportUC)
	exportDownloadHandler := export_download.NewHandler(getExportDownloadUC)

	authMiddleware := middleware.NewAuth(sessionMgr)
	csrfMiddleware := middleware.NewCSRF(cfg)
	sentryUserContextMW := middleware.NewSentryUserContext(profileRepo)

	// SentryのHTTPミドルウェア。Repanic: trueにより、panicをSentryに送った
	// あと再panicさせて、後続のRecovererに処理を委ねる。
	sentryHTTPHandler := sentryhttp.New(sentryhttp.Options{Repanic: true})

	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.RequestID)
	// クライアントIPはinternal/clientip (CF-Connecting-IP優先) で都度解決するため、
	// chiのIPミドルウェアは意図的に登録しない。chiのRealIPはIP spoofing
	// (GHSA-3fxj-6jh8-hvhx) のためdeprecatedでもあり、再導入しないこと。

	// Recovererをouter (chiのUseでは最初に登録 = チェーンの一番外側) に置く。
	// chiのRecovererはhttp.ErrAbortHandler以外をrecoverして500を書き、再panic
	// しないため、sentryhttpより「外側」(= panic伝搬の最後に届く位置) に置かないと、
	// sentryhttpがpanicを見られない。
	r.Use(chimiddleware.Recoverer)
	// sentryhttpはRecovererより「あとに登録」= chiのチェーンではinnermost
	// (= handlerに近い側)。handlerのpanicをsentryhttpのdeferがまず捕捉し、
	// Sentryに送信したあとRepanic: trueで再panicする。再panicはouterの
	// Recovererに到達し、Recovererが500レスポンスを書いて終了する。
	r.Use(sentryHTTPHandler.Handle)
	// chiのルートパターン (例: "/users/{id}") をSentryのトランザクション名に
	// 上書きする。sentryhttpより「あとに登録」することで、defer (LIFO) のタイミングで
	// sentryhttpのtransaction.Finish() より先にNameを確定できる。
	r.Use(middleware.SentryTransaction)

	if cfg.RailsAppURL != "" {
		proxyMiddleware, err := middleware.NewReverseProxyMiddleware(cfg.RailsAppURL, cfg, featureFlagRepo)
		if err != nil {
			slog.Error("リバースプロキシミドルウェアの初期化に失敗しました", "error", err)
			os.Exit(1)
		}
		r.Use(proxyMiddleware.Middleware)
	}

	// Go版が処理するルートのリクエストボディサイズを制限する。reverse_proxyより
	// 後に配置することでRails版にプロキシされるリクエストには影響させず、フォームを
	// パースするミドルウェア / ハンドラー (CSRF、MethodOverride、各ハンドラー) より前に置く。
	r.Use(middleware.BodyLimit)

	// i18nミドルウェアはAccept-Languageヘッダーからctxにロケールをセットする。
	// reverse_proxyより後に配置することで、Rails版にプロキシされるリクエストには
	// 走らせない。
	r.Use(i18n.Middleware)

	r.Use(flashMgr.Middleware)

	r.NotFound(httperror.NotFound)

	// 静的ファイルを配信する (Tailwind CLIとesbuildのビルド結果)。
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "ヘルスチェック: データベース接続エラー", "error", err)
			http.Error(w, "DB connection failed", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	r.Get("/manifest.json", manifestHandler.Show)

	r.Group(func(r chi.Router) {
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireNoAuth)
		r.Use(sentryUserContextMW.Middleware)
		r.Get("/sign_in", signInHandler.New)
		r.Post("/sign_in", signInHandler.Create)
		r.Get("/sign_up", signUpHandler.New)
		r.Post("/sign_up", signUpHandler.Create)
	})

	// ログアウト (認証済みユーザーのみ)。ログアウトフォームはGo版の /settings
	// ページが供給し、method="POST" で送信するため、リクエストはPOSTで本サーバーに
	// 届く。DELETEも同じハンドラーに登録しており、これは /passwordがPATCHとHTML
	// フォーム用のPOSTを二重登録しているのと同じ扱いである。
	//
	// CSRFミドルウェアがこのエンドポイントを保護する。/settingsページも同じCSRF
	// ミドルウェア配下で動くため、ページ読み込み時にトークンCookieが発行され、
	// ログアウトフォームには一致するトークンが埋め込まれる。
	r.Group(func(r chi.Router) {
		r.Use(middleware.PrivateCache)
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireAuth)
		r.Use(sentryUserContextMW.Middleware)
		r.Delete("/sign_out", signOutHandler.Delete)
		r.Post("/sign_out", signOutHandler.Delete)
	})

	r.Group(func(r chi.Router) {
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireNoAuth)
		r.Use(sentryUserContextMW.Middleware)

		r.Get("/password_reset", passwordResetHandler.New)
		r.Post("/password_reset", passwordResetHandler.Create)

		r.Get("/email_confirmation", emailConfirmationHandler.New)
		r.Post("/email_confirmation", emailConfirmationHandler.Create)

		r.Get("/password/edit", passwordHandler.Edit)
		// PATCHとPOSTを同じハンドラーへ登録している。パスワード編集フォームは
		// HTMLフォームで、method="POST" に _method=PATCHを添えて送信するため、
		// リクエストはPOSTで本サーバーに届く。
		r.Patch("/password", passwordHandler.Update)
		r.Post("/password", passwordHandler.Update)

		r.Get("/accounts/new", accountHandler.New)
		r.Post("/accounts", accountHandler.Create)
	})

	// 新規投稿フォーム・送信・リンクカードのフラグメント (認証済みユーザーのみ)。
	// リバースプロキシはこれらのパスをgoHandledPatterns (完全一致 + メソッド) で
	// 無条件にGo版へ振り分けるため、もうフィーチャーフラグの配下にはない。
	// POST /postsとPOST /linksは本来のPOSTのためMethod Overrideは不要で、
	// ルートを直接登録する。GET /links/newはフラグメントがPOST /linksフォーム用の
	// CSRFトークンを埋め込めるようCSRFミドルウェア配下に置く。
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

	// 設定系ページ (認証済みユーザーのみ)。CSRFミドルウェアはログアウトと
	// エクスポートのフォームへトークンを渡し、RequireAuthはnavbarとエクスポート
	// 画面の認可が使う現在のユーザーとプロフィールを渡す。
	r.Group(func(r chi.Router) {
		r.Use(middleware.PrivateCache)
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireAuth)
		r.Use(sentryUserContextMW.Middleware)

		r.Get("/settings", settingHandler.Index)
		r.Get("/settings/export", exportHandler.Show)
		r.Post("/settings/export", exportHandler.Create)
		r.Get("/settings/export/download", exportDownloadHandler.Show)
	})

	// ループバックではなく0.0.0.0でリッスンする。Dockerコンテナ内で動かす
	// 場合、ループバックにバインドしたリスナーには外から到達できないため。
	addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	slog.Info("HTTPサーバーを起動します", "addr", addr)

	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		slog.Info("シャットダウンシグナルを受信しました。サーバーを停止します...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := workerClient.Stop(shutdownCtx); err != nil {
			slog.Error("Workerの停止に失敗しました", "error", err)
		} else {
			slog.Info("Workerを停止しました")
		}

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("サーバーのシャットダウンに失敗しました", "error", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("サーバーの起動に失敗しました", "error", err)
		os.Exit(1)
	}

	slog.Info("サーバーが正常に停止しました")
}

// newEmailSenderはアプリのメールを配信するsenderを返す。プロバイダーが未設定の
// 場合はメールを破棄するsenderを返す。すべての種類のメールが同じインスタンスを通るため、
// そのデプロイがメールを配信するかどうかの判断は1度だけ行われる。
func newEmailSender(cfg *config.Config) email.Sender {
	if cfg.ResendAPIKey == "" || cfg.EmailFrom == "" {
		slog.Warn("Resend APIキーまたは送信元メールアドレスが設定されていないため、メール送信は無効です")
		return email.NewDiscardSender()
	}

	slog.Info("Resendクライアントを初期化しました")
	return email.NewResendSender(cfg.ResendAPIKey, cfg.EmailFrom, cfg.EmailFromName)
}

// newExportStorageはエクスポート機能が読み書きするオブジェクトストレージを
// 返す。storageReadyがfalseの場合はnilを返す。このフラグは呼び出し側が
// cfg.S3Readinessから一度だけ解決し、R2へ到達するすべてへ同じストレージを渡す
// ため、Workerとダウンロードのルートが別々のバケットを相手にするデプロイは
// 生じ得ない。
//
// nilは、それを生んだreadinessに対して安全である。このストレージを保持する
// UseCaseはいずれも同じフラグを受け取り、falseのときはストレージへ到達する前に
// 返るためである。具象型ではなくinterface型で返すことでこのnilをnil interfaceに
// 保ち、誤って呼び出した場合は資格情報の無いクライアントで通信するのではなく、
// その場で失敗する。
func newExportStorage(cfg *config.Config, storageReady bool) usecase.ExportObjectStorage {
	if !storageReady {
		return nil
	}

	return storage.NewS3ExportStorage(storage.S3Config{
		BucketName:      cfg.S3BucketName,
		Endpoint:        cfg.S3Endpoint,
		AccessKeyID:     cfg.S3AccessKeyID,
		SecretAccessKey: cfg.S3SecretAccessKey,
		Region:          cfg.S3Region,
	})
}

// newExportUsecasesはWorkerが実行するエクスポート系UseCaseを構築する。
// storageReadyがfalseの場合はゼロ値を返す。このフラグは呼び出し側が
// cfg.S3Readinessから一度だけ解決するため、ここで登録するWorkerとエクスポート
// 画面は、乖離しうる2つの式ではなく同じ値でゲートされる。exportStorageは
// newExportStorageが同じフラグから構築したストレージであり、これらのUseCaseを
// 構築するときにちょうど非nilになる。ゼロ値を返すことで、
// MEWST_S3_* を1つも設定しないままフィーチャーフラグOFFの状態をデプロイできる
// (エクスポート系Workerも定期ジョブも登録されず、エクスポートの処理は動作しない)。
// 一部だけ設定された状態はconfig.Loadが起動時に拒否するため、ここには到達しない。
//
// オブジェクトストレージに触れないのはリコンシリエーションだけだが、これらはまとめて
// 構築する。ストレージが無ければエクスポート行自体を作成できないため、
// リコンシリエーションは存在し得ない処理を回復することになるからである。
func newExportUsecases(
	cfg *config.Config,
	storageReady bool,
	exportStorage usecase.ExportObjectStorage,
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
