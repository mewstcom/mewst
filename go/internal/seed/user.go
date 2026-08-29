package seed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// historyMonths is how far back the accounts that hold a history reach. The
// three years of posts are generated across this span, so a profile that is to
// hold them has to have joined before the oldest of them: a profile that
// joined today while its posts go back three years is a state no screen can
// arrive at.
//
// [Ja] historyMonths は、履歴を持つアカウントがどこまで遡るか。3 年分のポストは
// この期間へ生成されるため、それらを持つプロフィールは最も古いポストより前に参加して
// いる必要がある。3 年前のポストを持ちながら今日参加したプロフィールは、どの画面から
// も辿り着けない状態であるため。
const historyMonths = 36

// seedAccount is one account a run created, as the rows that make it up.
//
// The generators that follow reach for these rather than reading the rows back
// out of the database: a post needs the profile that wrote it, a follow needs
// both actors, and a feature flag needs the actor it is granted to.
//
// [Ja] seedAccount は、実行が作成したアカウント 1 件を、それを構成する行として
// 表したもの。
//
// 後続の生成器は、行をデータベースから読み直すのではなくこれを参照する。ポストは
// それを書いたプロフィールを、フォローは双方の actor を、フィーチャーフラグはそれを
// 付与する actor を必要とするため。
type seedAccount struct {
	// roster is the entry the account was created from. It travels with the
	// rows so that a run can report the account by what it is there to show,
	// which is written in the roster and not in any of the rows.
	//
	// [Ja] roster はアカウントの作成元になった名簿の 1 件。行と一緒に持ち回るのは、
	// 実行がアカウントを「何を確認するためにいるのか」で報告できるようにするため。
	// それは名簿に書かれており、どの行にも入らない。
	roster rosterUser

	user    *model.User
	profile *model.Profile
	actor   *model.Actor
}

// roleProfile is what the seed decides about a role's profile, as against what
// the roster says about the account.
//
// The roster says who exists and what each account is there to show. How long
// an account has been here, and what its profile says about itself, is part of
// the data the seed generates, so it stays beside the generators that depend
// on it rather than moving into the file.
//
// [Ja] roleProfile は、名簿がアカウントについて述べることに対して、シードが役割の
// プロフィールについて決めること。
//
// 名簿が述べるのは、誰がいて、それぞれが何を確認するためにいるのかまで。アカウントが
// いつからいるのか、プロフィールが自身について何と書いているのかは、シードが生成する
// データの一部であり、ファイルへ動かさず、それに依存する生成器のかたわらに置く。
type roleProfile struct {
	// joinedMonthsAgo is how far back joined_at is placed, counted from the
	// instant the run is anchored to.
	//
	// [Ja] joinedMonthsAgo は、実行が基準とする時点から数えて、joined_at を
	// どれだけ過去へ置くか。
	joinedMonthsAgo int

	// description is the profile's self-introduction. Every profile holds one
	// so that the screens that show it are not looked at in a state no account
	// in production is in.
	//
	// [Ja] description はプロフィールの自己紹介。すべてのプロフィールがこれを持つ
	// のは、それを表示する画面を、本番のどのアカウントも取らない状態で見ることに
	// ならないようにするため。
	description string
}

// roleProfiles holds one entry per role in allSeedRoles. The joined dates are
// staggered rather than shared: a list of profiles that all joined in the same
// month cannot show whether the date being displayed is the profile's own.
//
// [Ja] roleProfiles は allSeedRoles の役割ごとに 1 件を持つ。参加日をずらして
// いるのは、すべてが同じ月に参加したプロフィールの一覧では、表示されている日付が
// そのプロフィール自身のものなのかを確認できないため。
var roleProfiles = map[seedRole]roleProfile{
	roleMain: {
		joinedMonthsAgo: historyMonths,
		description:     "毎日のできごとを書き留めています。コーヒーと散歩と、ときどき写真。",
	},
	roleFollower: {
		joinedMonthsAgo: 30,
		description:     "近所のパン屋とサウナの話が多めです。",
	},
	roleEnglish: {
		joinedMonthsAgo: 24,
		description:     "日々のことを英語で書いています。",
	},
	roleNewcomer: {
		// A newcomer joined just now: the screens this role is for are the
		// ones seen right after signing up, and a join date in the past would
		// not be one of them.
		//
		// [Ja] newcomer は今まさに参加した。この役割が対象とする画面はサインアップ
		// 直後に見えるものであり、過去の参加日はそこに含まれない。
		joinedMonthsAgo: 0,
		description:     "はじめました。",
	},
	roleDiscarded: {
		joinedMonthsAgo: 12,
		description:     "山とカメラの話をしています。",
	},
}

// accountRepositories are the repositories one account is written through.
// They are gathered into one value rather than passed one by one, so that the
// four writes that make up an account read as one step at the call site.
//
// [Ja] accountRepositories は、アカウント 1 件を書き込むためのリポジトリ。1 つずつ
// 渡すのではなく 1 つの値にまとめるのは、アカウントを構成する 4 つの書き込みが、
// 呼び出し側で 1 つの手順として読めるようにするため。
type accountRepositories struct {
	profile     *repository.ProfileRepository
	user        *repository.UserRepository
	userProfile *repository.UserProfileRepository
	actor       *repository.ActorRepository
}

// newAccountRepositories binds the repositories to the run's transaction.
//
// [Ja] newAccountRepositories は、リポジトリを実行のトランザクションに束ねる。
func newAccountRepositories(tx *sql.Tx) accountRepositories {
	q := query.New(tx)

	return accountRepositories{
		profile:     repository.NewProfileRepository(q),
		user:        repository.NewUserRepository(q),
		userProfile: repository.NewUserProfileRepository(q),
		actor:       repository.NewActorRepository(q),
	}
}

// createAccounts creates the accounts the roster names, in the order it names
// them, and returns them for the generators that follow.
//
// [Ja] createAccounts は、名簿が挙げるアカウントを、名簿が挙げる順に作成し、後続の
// 生成器のために返す。
func createAccounts(ctx context.Context, tx *sql.Tx, roster *userRoster, now time.Time) ([]seedAccount, error) {
	repos := newAccountRepositories(tx)

	accounts := make([]seedAccount, 0, len(roster.users))
	for _, entry := range roster.users {
		account, err := createAccount(ctx, tx, repos, entry, roster.passwordDigest, now)
		if err != nil {
			// The role is named rather than the atname: it is what the roster
			// entry is found by, and what the generators refer to the account
			// as.
			//
			// [Ja] atname ではなく役割を名指しする。名簿の該当箇所を見つけるのに
			// 使うのも、生成器がそのアカウントを指すのに使うのも役割であるため。
			return nil, fmt.Errorf("役割 %s のアカウントの作成に失敗: %w", entry.role, err)
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

// createAccount writes the four rows one account is made of, grants it the
// feature flags the roster gave it, and discards its profile if that is what
// its role is there to show.
//
// The rows are written in the order account creation writes them, through the
// same repositories, so that an account the seed created is one the
// application could have created.
//
// [Ja] createAccount は、アカウントを構成する 4 つの行を書き、名簿が与えた
// フィーチャーフラグを付与し、役割が示すべきものがそれであればプロフィールを削除済みに
// する。
//
// 行はアカウント作成が書き込むのと同じ順序で、同じリポジトリを通して書く。シードが
// 作成したアカウントが、アプリケーションが作成しえたアカウントであるようにするため。
func createAccount(
	ctx context.Context,
	tx *sql.Tx,
	repos accountRepositories,
	entry rosterUser,
	passwordDigest string,
	now time.Time,
) (seedAccount, error) {
	shape := roleProfiles[entry.role]

	profile, err := repos.profile.Create(ctx, repository.CreateProfileInput{
		OwnerType:     model.ProfileOwnerTypeUser,
		Atname:        entry.atname,
		Name:          entry.name,
		Description:   shape.description,
		ImageURL:      "",
		JoinedAt:      now.AddDate(0, -shape.joinedMonthsAgo, 0),
		AvatarKind:    usecase.DefaultAvatarKind,
		GravatarEmail: "",
		GravatarURL:   "",
	})
	if err != nil {
		return seedAccount{}, fmt.Errorf("プロフィールの作成に失敗: %w", err)
	}

	// The digest is the one the roster prepared: the shared password was
	// hashed once while the roster was read, and the plaintext was not carried
	// this far.
	//
	// [Ja] ダイジェストは名簿が用意したもの。共通パスワードは名簿の読み込み時に
	// 一度だけハッシュ化されており、平文はここまで持ち越されていない。
	user, err := repos.user.Create(ctx, repository.CreateUserInput{
		Email:          entry.email,
		PasswordDigest: passwordDigest,
		Locale:         entry.locale,
		TimeZone:       entry.timeZone,
	})
	if err != nil {
		return seedAccount{}, fmt.Errorf("ユーザーの作成に失敗: %w", err)
	}

	if _, err := repos.userProfile.Create(ctx, repository.CreateUserProfileInput{
		UserID:    user.ID,
		ProfileID: profile.ID,
	}); err != nil {
		return seedAccount{}, fmt.Errorf("ユーザープロフィール関連付けの作成に失敗: %w", err)
	}

	actor, err := repos.actor.Create(ctx, repository.CreateActorInput{
		UserID:    user.ID,
		ProfileID: profile.ID,
	})
	if err != nil {
		return seedAccount{}, fmt.Errorf("アクターの作成に失敗: %w", err)
	}

	if err := createFeatureFlags(ctx, tx, actor.ID, entry.featureFlags); err != nil {
		return seedAccount{}, err
	}

	if entry.role == roleDiscarded {
		if err := discardProfile(ctx, tx, profile, now); err != nil {
			return seedAccount{}, err
		}
	}

	return seedAccount{
		roster:  entry,
		user:    user,
		profile: profile,
		actor:   actor,
	}, nil
}

// createFeatureFlags grants the account the flags the roster gave it.
//
// A flag is a row per actor rather than a column on one, so the grant is
// written here: the application only ever asks whether a flag is on, and has
// no reason to hold a way to turn one on for somebody.
//
// [Ja] createFeatureFlags は、名簿が与えたフラグをアカウントへ付与する。
//
// フラグは actor ごとの行であり、actor が持つカラムではないため、付与はここで書く。
// アプリケーションが行うのはフラグが有効かを尋ねることだけで、誰かに対してフラグを
// 有効にする手段を持つ理由が無いためである。
func createFeatureFlags(ctx context.Context, tx *sql.Tx, actorID model.ActorID, names []model.FeatureFlagName) error {
	for _, name := range names {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO feature_flags (actor_id, name)
			VALUES ($1, $2)
		`, uuid.UUID(actorID), string(name)); err != nil {
			return fmt.Errorf("フィーチャーフラグ %s の付与に失敗: %w", name, err)
		}
	}

	return nil
}

// discardProfile marks the profile as deleted, which is the state the
// discarded role is there to show.
//
// The write is here rather than behind a repository method: deleting a profile
// is not something the Go version does yet, and a method that only the seed
// calls would be a way into that state that production never takes.
//
// The model is brought along with the row it stands for, so that a generator
// that receives this account is not handed a profile that says it is still
// there.
//
// [Ja] discardProfile は、プロフィールを削除済みにする。discarded 役割が示すために
// いるのがその状態である。
//
// この書き込みをリポジトリのメソッドではなくここに置くのは、プロフィールの削除が
// Go 版のまだ行わない操作であり、シードだけが呼ぶメソッドは、本番が通らない経路を
// その状態へ向けて 1 つ開けることになるため。
//
// モデルは、それが表す行と一緒に更新する。このアカウントを受け取った生成器へ、まだ
// 存在していると告げるプロフィールを渡さないようにするため。
func discardProfile(ctx context.Context, tx *sql.Tx, profile *model.Profile, discardedAt time.Time) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE profiles
		SET discarded_at = $2, updated_at = NOW()
		WHERE id = $1
	`, uuid.UUID(profile.ID), discardedAt); err != nil {
		return fmt.Errorf("プロフィールの削除済み化に失敗: %w", err)
	}

	profile.DiscardedAt = &discardedAt

	return nil
}
