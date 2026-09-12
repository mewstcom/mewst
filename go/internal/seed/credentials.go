package seed

import (
	"fmt"
	"os"
	"slices"
)

// readsTheRosterPassword is why a credentials lookup is confined to a
// development environment. Nothing here is destructive, but the roster
// describes the accounts of one development environment, and its password is
// not something to hand to a process that is not running in one.
//
// [Ja] readsTheRosterPassword は、資格情報の取得を開発環境に限っている理由。
// ここでの処理は破壊的ではないが、名簿はある開発環境のアカウントを記述したもので
// あり、そのパスワードは、そこで動いているのではないプロセスへ渡すものではない。
const readsTheRosterPassword = "開発用ユーザーの名簿が持つパスワードを出力するため"

// Credentials is what one seeded account signs in with.
//
// The password is the shared plaintext the roster carries rather than the
// digest the accounts store, because this is read by a caller that is about to
// type it into the sign-in form.
//
// [Ja] Credentials は、シードが作成したアカウント 1 件がサインインに使うもの。
//
// パスワードは、アカウントが保存するダイジェストではなく名簿が持つ共通の平文で
// ある。これを読むのは、これからサインインフォームへ入力しようとしている
// 呼び出し側であるため。
type Credentials struct {
	Email    string
	Password string
}

// DevCredentials returns the credentials of the account the roster gives role.
//
// It exists because the browser verification script signs in as an account the
// seed created, and bash cannot read TOML. Both sides reaching the same roster
// through this function is what keeps a change to the roster from breaking the
// sign-in that follows a seed run.
//
// [Ja] DevCredentials は、名簿が role に与えているアカウントの資格情報を返す。
//
// ブラウザ確認のスクリプトがサインインするのはシードが作成したアカウントであり、
// bash は TOML を読めないため、この関数がある。双方がこの関数を通して同じ名簿へ
// 辿り着くことが、名簿への変更がシード実行後のサインインを壊さない理由になる。
func DevCredentials(role string) (Credentials, error) {
	return devCredentials(os.Getenv(appEnvVar), rosterPath, role)
}

// devCredentials looks the role named name up in the roster at path.
//
// The environment and the path are taken as parameters rather than read here,
// for the same reason Runner takes them: what the lookup decides on is then
// visible to a test that does not have to set a variable the rest of the test
// binary shares.
//
// The environment is checked first, before the roster is read. Nothing here is
// destructive, but a roster describes the accounts of one development
// environment, and a process that is not running in one has no business being
// handed a password from it.
//
// [Ja] devCredentials は、path の名簿から name の役割を引く。
//
// 環境とパスを自分で読まずに仮引数で受け取るのは Runner と同じ理由による。
// 何を見て判断しているのかが、テストバイナリ全体で共有される変数を設定せずに
// 済むテストからも見えるようになる。
//
// 環境の検査は、名簿を読むより前に最初に行う。ここでの処理はいずれも破壊的では
// ないが、名簿はある開発環境のアカウントを記述したものであり、そこで動いている
// のではないプロセスへ、そこのパスワードを渡す理由が無いため。
func devCredentials(env, path, name string) (Credentials, error) {
	if err := requireDevEnvironment(env, readsTheRosterPassword); err != nil {
		return Credentials{}, err
	}

	role, err := lookUpSignInRole(name)
	if err != nil {
		return Credentials{}, err
	}

	// The roster is read without hashing the password, which loadUserRoster
	// does on its way to the digest the accounts are written with. What is
	// wanted here is the plaintext itself, and the checks that decide whether
	// the roster can be used at all are the same either way.
	//
	// [Ja] 名簿はパスワードをハッシュ化せずに読む。loadUserRoster がハッシュ化
	// するのは、アカウントを書き込むダイジェストへ辿り着くためである。ここで必要
	// なのは平文そのものであり、名簿を使えるかどうかを決める検査はどちらでも同じ
	// ものが働く。
	file, err := decodeRosterFile(path)
	if err != nil {
		return Credentials{}, err
	}

	users, err := file.validate()
	if err != nil {
		return Credentials{}, fmt.Errorf("開発用ユーザーの名簿 %s: %w", path, err)
	}

	for _, entry := range users {
		if entry.role == role {
			return Credentials{Email: entry.email, Password: file.Password}, nil
		}
	}

	// validate has already refused a roster that is missing a role, so this is
	// reached only by a role that was added to the generators without being
	// added to that check.
	//
	// [Ja] 役割を欠いた名簿は validate がすでに拒否しているため、ここへ辿り着く
	// のは、その検査へ足されないまま生成器へ足された役割だけになる。
	return Credentials{}, fmt.Errorf("役割 %s の [[users]] が名簿 %s にありません", role, path)
}

// lookUpSignInRole turns the requested name into a role that can be signed in
// as.
//
// A role the roster holds but nobody can sign in as is refused as such rather
// than as an unknown name, so that the answer says why this particular account
// is not one to sign in with.
//
// [Ja] lookUpSignInRole は、要求された名前を、サインインできる役割へ変換する。
//
// 名簿は持っているが誰もそれではサインインできない役割は、未知の名前としてでは
// なくそのものとして拒否する。この特定のアカウントがなぜサインインするための
// ものではないのかを、応答が述べるようにするため。
func lookUpSignInRole(name string) (seedRole, error) {
	role := seedRole(name)

	if role == roleDiscarded {
		return "", fmt.Errorf(
			"役割 %s は削除済みプロフィールのアカウントのため、サインインできません。指定できるのは %s です",
			role, joinSeedRoles(signInRoles()),
		)
	}
	if !slices.Contains(allSeedRoles, role) {
		return "", fmt.Errorf("役割 %q は名簿が持たない役割です。指定できるのは %s です", name, joinSeedRoles(signInRoles()))
	}

	return role, nil
}

// signInRoles lists the roles an account can be signed in as, which is every
// role the roster holds but the discarded one. A discarded profile that can
// sign in is a state production never reaches, so the seed does not offer one.
//
// [Ja] signInRoles は、そのアカウントでサインインできる役割の一覧。名簿が持つ
// 役割から削除済みのものを除いた全件になる。削除済みプロフィールでサインイン
// できる状態は本番では起こり得ないため、シードはそれを提供しない。
func signInRoles() []seedRole {
	roles := make([]seedRole, 0, len(allSeedRoles))
	for _, role := range allSeedRoles {
		if role == roleDiscarded {
			continue
		}
		roles = append(roles, role)
	}

	return roles
}
