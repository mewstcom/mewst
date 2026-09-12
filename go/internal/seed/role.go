package seed

// seedRole is the logical name by which a generator asks for an account. A
// generator names the account three years of posts belong to, or the account
// that follows it, never the first or the second account, so that an account
// added to the seed leaves the generators that do not need its role untouched.
//
// [Ja] seedRole は、生成器がアカウントを求めるときに使う論理名。生成器が名指し
// するのは「1 人目」「2 人目」ではなく、3 年分のポストを持つアカウントや、それを
// フォローしているアカウントになる。シードにアカウントを 1 つ足したとき、その役割を
// 必要としない生成器が変わらないようにするため。
type seedRole string

const (
	// roleMain is the account the development environment is mostly looked at
	// through. It holds the three years of posts the export is checked
	// against, it holds every feature flag, and it is the account
	// make browse-login signs in as unless another role is named.
	//
	// [Ja] roleMain は開発環境をおもに見るためのアカウント。エクスポートの確認に
	// 使う 3 年分のポストとフィーチャーフラグを全件持ち、別の役割を指定しない
	// かぎり make browse-login がサインインするのもこのアカウントになる。
	roleMain seedRole = "main"

	// roleFollower follows roleMain and is followed back, which is what gives
	// both of them a home timeline that is not empty. It holds no feature
	// flag, so a screen can be compared with roleMain's before and after a
	// flag.
	//
	// [Ja] roleFollower は roleMain と相互フォローの関係にあり、それが両者の
	// ホームタイムラインを空でない状態にする。フィーチャーフラグを 1 つも持たない
	// ため、フラグの有無による画面の違いを roleMain と見比べられる。
	roleFollower seedRole = "follower"

	// roleEnglish reads the application in English from a UTC clock. The
	// export HTML picks its wording, its date format and its month boundaries
	// from the locale and the time zone of the account it belongs to, so this
	// role is what makes those visible next to roleMain's.
	//
	// [Ja] roleEnglish は UTC の時計で英語のアプリケーションを読むアカウント。
	// エクスポートの HTML は、文言・日時表記・月境界を、そのアカウントのロケールと
	// タイムゾーンから出し分ける。この役割があることで、それらを roleMain のものと
	// 並べて確認できる。
	roleEnglish seedRole = "english"

	// roleNewcomer holds no post at all while holding every feature flag. The
	// two are set independently on purpose: an account that reaches the Go
	// version of a screen but has written nothing is the only way to see an
	// empty export, and the screens as they look just after signing up.
	//
	// [Ja] roleNewcomer はポストを 1 件も持たないまま、フィーチャーフラグを全件
	// 持つ。この 2 つを独立に組み合わせているのは意図的で、画面の Go 版へ辿り着ける
	// のに何も書いていないアカウントは、空のエクスポートと、サインアップ直後の画面を
	// 見るための唯一の手段であるため。
	roleNewcomer seedRole = "newcomer"

	// roleDiscarded is the account whose profile has been deleted
	// (profiles.discarded_at). It is what shows how a post whose author is
	// gone appears to everyone else. It is not a role to sign in as: a
	// discarded profile that can sign in is a state production never reaches.
	//
	// [Ja] roleDiscarded はプロフィールが削除済み (profiles.discarded_at) の
	// アカウント。作者が居なくなったポストが他の人にどう見えるのかを示す。サイン
	// インするための役割ではない。削除済みプロフィールでサインインできる状態は、
	// 本番では起こり得ないため。
	roleDiscarded seedRole = "discarded"
)

// allSeedRoles lists every role a generator names. The roster has to hold one
// account for each of them, which is what lets a generator ask for a role and
// get an account back.
//
// [Ja] allSeedRoles は生成器が名指しする役割の一覧。名簿はこのそれぞれに 1 件ずつ
// アカウントを持つ必要があり、それによって生成器は役割を求めてアカウントを
// 受け取れる。
var allSeedRoles = []seedRole{roleMain, roleFollower, roleEnglish, roleNewcomer, roleDiscarded}
