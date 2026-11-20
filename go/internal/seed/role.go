package seed

// seedRoleは、生成器がアカウントを求めるときに使う論理名。生成器が名指し
// するのは「1人目」「2人目」ではなく、3年分のポストを持つアカウントや、それを
// フォローしているアカウントになる。シードにアカウントを1つ足したとき、その役割を
// 必要としない生成器が変わらないようにするため。
type seedRole string

const (
	// roleMainは開発環境をおもに見るためのアカウント。エクスポートの確認に
	// 使う3年分のポストとフィーチャーフラグを全件持ち、別の役割を指定しない
	// かぎりmake browse-loginがサインインするのもこのアカウントになる。
	roleMain seedRole = "main"

	// roleFollowerはroleMainと相互フォローの関係にあり、それが両者の
	// ホームタイムラインを空でない状態にする。フィーチャーフラグを1つも持たない
	// ため、フラグの有無による画面の違いをroleMainと見比べられる。
	roleFollower seedRole = "follower"

	// roleEnglishはUTCの時計で英語のアプリケーションを読むアカウント。
	// エクスポートのHTMLは、文言・日時表記・月境界を、そのアカウントのロケールと
	// タイムゾーンから出し分ける。この役割があることで、それらをroleMainのものと
	// 並べて確認できる。
	roleEnglish seedRole = "english"

	// roleNewcomerはポストを1件も持たないまま、フィーチャーフラグを全件
	// 持つ。この2つを独立に組み合わせているのは意図的で、画面のGo版へ辿り着ける
	// のに何も書いていないアカウントは、空のエクスポートと、サインアップ直後の画面を
	// 見るための唯一の手段であるため。
	roleNewcomer seedRole = "newcomer"

	// roleDiscardedはプロフィールが削除済み (profiles.discarded_at) の
	// アカウント。作者が居なくなったポストが他の人にどう見えるのかを示す。サイン
	// インするための役割ではない。削除済みプロフィールでサインインできる状態は、
	// 本番では起こり得ないため。
	roleDiscarded seedRole = "discarded"
)

// allSeedRolesは生成器が名指しする役割の一覧。名簿はこのそれぞれに1件ずつ
// アカウントを持つ必要があり、それによって生成器は役割を求めてアカウントを
// 受け取れる。
var allSeedRoles = []seedRole{roleMain, roleFollower, roleEnglish, roleNewcomer, roleDiscarded}
