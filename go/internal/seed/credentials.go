package seed

import (
	"fmt"
	"os"
	"slices"
)

// readsTheRosterPasswordは、資格情報の取得を開発環境に限っている理由。
// ここでの処理は破壊的ではないが、名簿はある開発環境のアカウントを記述したもので
// あり、そのパスワードは、そこで動いているのではないプロセスへ渡すものではない。
const readsTheRosterPassword = "開発用ユーザーの名簿が持つパスワードを出力するため"

// Credentialsは、シードが作成したアカウント1件がサインインに使うもの。
//
// パスワードは、アカウントが保存するダイジェストではなく名簿が持つ共通の平文で
// ある。これを読むのは、これからサインインフォームへ入力しようとしている
// 呼び出し側であるため。
type Credentials struct {
	Email    string
	Password string
}

// DevCredentialsは、名簿がroleに与えているアカウントの資格情報を返す。
//
// ブラウザ確認のスクリプトがサインインするのはシードが作成したアカウントであり、
// bashはTOMLを読めないため、この関数がある。双方がこの関数を通して同じ名簿へ
// 辿り着くことが、名簿への変更がシード実行後のサインインを壊さない理由になる。
func DevCredentials(role string) (Credentials, error) {
	return devCredentials(os.Getenv(appEnvVar), rosterPath, role)
}

// devCredentialsは、pathの名簿からnameの役割を引く。
//
// 環境とパスを自分で読まずに仮引数で受け取るのはRunnerと同じ理由による。
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

	// 名簿はパスワードをハッシュ化せずに読む。loadUserRosterがハッシュ化
	// するのは、アカウントを書き込むダイジェストへ辿り着くためである。ここで必要
	// なのは平文そのものであり、名簿を使えるかどうかを決める検査はどちらでも同じ
	// ものが働く。
	file, err := decodeRosterFile(path)
	if err != nil {
		return Credentials{}, err
	}

	users, err := file.validate()
	if err != nil {
		return Credentials{}, fmt.Errorf("開発用ユーザーの名簿%s: %w", path, err)
	}

	for _, entry := range users {
		if entry.role == role {
			return Credentials{Email: entry.email, Password: file.Password}, nil
		}
	}

	// 役割を欠いた名簿はvalidateがすでに拒否しているため、ここへ辿り着く
	// のは、その検査へ足されないまま生成器へ足された役割だけになる。
	return Credentials{}, fmt.Errorf("役割%sの [[users]] が名簿%sにありません", role, path)
}

// lookUpSignInRoleは、要求された名前を、サインインできる役割へ変換する。
//
// 名簿は持っているが誰もそれではサインインできない役割は、未知の名前としてでは
// なくそのものとして拒否する。この特定のアカウントがなぜサインインするための
// ものではないのかを、応答が述べるようにするため。
func lookUpSignInRole(name string) (seedRole, error) {
	role := seedRole(name)

	if role == roleDiscarded {
		return "", fmt.Errorf(
			"役割%sは削除済みプロフィールのアカウントのため、サインインできません。指定できるのは%sです",
			role, joinSeedRoles(signInRoles()),
		)
	}
	if !slices.Contains(allSeedRoles, role) {
		return "", fmt.Errorf("役割%qは名簿が持たない役割です。指定できるのは%sです", name, joinSeedRoles(signInRoles()))
	}

	return role, nil
}

// signInRolesは、そのアカウントでサインインできる役割の一覧。名簿が持つ
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
