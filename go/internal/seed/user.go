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

// historyMonthsは、履歴を持つアカウントがどこまで遡るか。3年分のポストは
// この期間へ生成されるため、それらを持つプロフィールは最も古いポストより前に参加して
// いる必要がある。3年前のポストを持ちながら今日参加したプロフィールは、どの画面から
// も辿り着けない状態であるため。
const historyMonths = 36

// seedAccountは、実行が作成したアカウント1件を、それを構成する行として
// 表したもの。
//
// 後続の生成器は、行をデータベースから読み直すのではなくこれを参照する。ポストは
// それを書いたプロフィールを、フォローは双方のactorを、フィーチャーフラグはそれを
// 付与するactorを必要とするため。
type seedAccount struct {
	// rosterはアカウントの作成元になった名簿の1件。行と一緒に持ち回るのは、
	// 実行がアカウントを「何を確認するためにいるのか」で報告できるようにするため。
	// それは名簿に書かれており、どの行にも入らない。
	roster rosterUser

	user    *model.User
	profile *model.Profile
	actor   *model.Actor
}

// roleProfileは、名簿がアカウントについて述べることに対して、シードが役割の
// プロフィールについて決めること。
//
// 名簿が述べるのは、誰がいて、それぞれが何を確認するためにいるのかまで。アカウントが
// いつからいるのか、プロフィールが自身について何と書いているのかは、シードが生成する
// データの一部であり、ファイルへ動かさず、それに依存する生成器のかたわらに置く。
type roleProfile struct {
	// joinedMonthsAgoは、実行が基準とする時点から数えて、joined_atを
	// どれだけ過去へ置くか。
	joinedMonthsAgo int

	// createdMinutesAgoは、行が書き込まれた時点から数えてcreated_atを
	// どれだけ過去へ置くか。おすすめフォローの順序を固定するのはこの値である。
	// おすすめを提示する画面はプロフィールをprofiles.created_atの降順で並べる。
	// 実行1回分のプロフィールはいずれも同じ時点で挿入されるため、このカラムに
	// 手を触れない実行では、行が返ってきた順序のまま並べられることになる。
	createdMinutesAgo int

	// descriptionはプロフィールの自己紹介。すべてのプロフィールがこれを持つ
	// のは、それを表示する画面を、本番のどのアカウントも取らない状態で見ることに
	// ならないようにするため。
	description string
}

// roleProfilesはallSeedRolesの役割ごとに1件を持つ。参加日をずらして
// いるのは、すべてが同じ月に参加したプロフィールの一覧では、表示されている日付が
// そのプロフィール自身のものなのかを確認できないため。
var roleProfiles = map[seedRole]roleProfile{
	roleMain: {
		joinedMonthsAgo:   historyMonths,
		createdMinutesAgo: 0,
		description:       "毎日のできごとを書き留めています。コーヒーと散歩と、ときどき写真。",
	},
	roleFollower: {
		joinedMonthsAgo:   30,
		createdMinutesAgo: 1,
		description:       "近所のパン屋とサウナの話が多めです。",
	},
	roleEnglish: {
		joinedMonthsAgo:   24,
		createdMinutesAgo: 2,
		description:       "日々のことを英語で書いています。",
	},
	roleNewcomer: {
		// newcomerは今まさに参加した。この役割が対象とする画面はサインアップ
		// 直後に見えるものであり、過去の参加日はそこに含まれない。
		joinedMonthsAgo: 0,
		description:     "はじめました。",
	},
	roleDiscarded: {
		joinedMonthsAgo: 12,
		description:     "山とカメラの話をしています。",
	},
}

// accountRepositoriesは、アカウント1件を書き込むためのリポジトリ。1つずつ
// 渡すのではなく1つの値にまとめるのは、アカウントを構成する4つの書き込みが、
// 呼び出し側で1つの手順として読めるようにするため。
type accountRepositories struct {
	profile     *repository.ProfileRepository
	user        *repository.UserRepository
	userProfile *repository.UserProfileRepository
	actor       *repository.ActorRepository
}

// newAccountRepositoriesは、リポジトリを実行のトランザクションに束ねる。
func newAccountRepositories(tx *sql.Tx) accountRepositories {
	q := query.New(tx)

	return accountRepositories{
		profile:     repository.NewProfileRepository(q),
		user:        repository.NewUserRepository(q),
		userProfile: repository.NewUserProfileRepository(q),
		actor:       repository.NewActorRepository(q),
	}
}

// createAccountsは、名簿が挙げるアカウントを、名簿が挙げる順に作成し、後続の
// 生成器のために返す。
func createAccounts(ctx context.Context, tx *sql.Tx, roster *userRoster, now time.Time) ([]seedAccount, error) {
	repos := newAccountRepositories(tx)

	accounts := make([]seedAccount, 0, len(roster.users))
	for _, entry := range roster.users {
		account, err := createAccount(ctx, tx, repos, entry, roster.passwordDigest, now)
		if err != nil {
			// atnameではなく役割を名指しする。名簿の該当箇所を見つけるのに
			// 使うのも、生成器がそのアカウントを指すのに使うのも役割であるため。
			return nil, fmt.Errorf("役割%sのアカウントの作成に失敗: %w", entry.role, err)
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

// accountForRoleは、roleのために作成されたアカウントを返す。
//
// 名簿は読み込み時に役割ごとの1件があることを検査しているため、ここで見つからない
// 役割は、名簿の書き誤りではなく、シードが持たない役割を生成器が求めていることを
// 意味する。想定せずに報告するのは、失敗がその役割を名指しするようにするため。そうで
// なければ、いくつも先の文で、存在しないプロフィールとして現れることになる。
func accountForRole(accounts []seedAccount, role seedRole) (seedAccount, error) {
	for _, account := range accounts {
		if account.roster.role == role {
			return account, nil
		}
	}

	return seedAccount{}, fmt.Errorf("役割%sのアカウントが作成されていません", role)
}

// createAccountは、アカウントを構成する4つの行を書き、名簿が与えた
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

	if err := backdateProfileCreatedAt(ctx, tx, profile, shape.createdMinutesAgo); err != nil {
		return seedAccount{}, err
	}

	// ダイジェストは名簿が用意したもの。共通パスワードは名簿の読み込み時に
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

// createFeatureFlagsは、名簿が与えたフラグをアカウントへ付与する。
//
// フラグはactorごとの行であり、actorが持つカラムではないため、付与はここで書く。
// アプリケーションが行うのはフラグが有効かを尋ねることだけで、誰かに対してフラグを
// 有効にする手段を持つ理由が無いためである。
func createFeatureFlags(ctx context.Context, tx *sql.Tx, actorID model.ActorID, names []model.FeatureFlagName) error {
	for _, name := range names {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO feature_flags (actor_id, name)
			VALUES ($1, $2)
		`, uuid.UUID(actorID), string(name)); err != nil {
			return fmt.Errorf("フィーチャーフラグ%sの付与に失敗: %w", name, err)
		}
	}

	return nil
}

// backdateProfileCreatedAtは、プロフィールのcreated_atを、行が書き込まれた
// 時点からminutesAgo分だけ過去へ動かす。
//
// このカラムは行の挿入時にデータベースの時計で書かれ、それは1つのトランザクション
// のどの行にとっても同じ時点になる。そのため、行が置かれた後に動かす。モデルも一緒に
// 動かすのは、生成器が受け渡すプロフィールが、書き込まれた行を記述しているようにする
// ため。
func backdateProfileCreatedAt(ctx context.Context, tx *sql.Tx, profile *model.Profile, minutesAgo int) error {
	if minutesAgo == 0 {
		return nil
	}

	createdAt := storedInstant(profile.CreatedAt.Add(-time.Duration(minutesAgo) * time.Minute))

	if _, err := tx.ExecContext(ctx, `
		UPDATE profiles
		SET created_at = $2
		WHERE id = $1
	`, uuid.UUID(profile.ID), createdAt); err != nil {
		return fmt.Errorf("プロフィール作成日時の設定に失敗: %w", err)
	}

	profile.CreatedAt = createdAt

	return nil
}

// discardProfileは、プロフィールを削除済みにする。discarded役割が示すために
// いるのがその状態である。
//
// この書き込みをリポジトリのメソッドではなくここに置くのは、プロフィールの削除が
// Go版のまだ行わない操作であり、シードだけが呼ぶメソッドは、本番が通らない経路を
// その状態へ向けて1つ開けることになるため。
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
