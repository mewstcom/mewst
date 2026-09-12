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
)

// amounts is how many rows of each kind a run creates. The counts sit together
// in one struct rather than beside the loops that read them, so that what a
// run produces can be read in one place, and so that a test can ask for the
// few rows it needs where a run asks for thousands.
//
// [Ja] amounts は、実行 1 回分が各種の行を何件作るか。件数を、それを読むループの
// かたわらではなく 1 つの構造体にまとめているのは、実行が何を作るのかを 1 箇所で
// 読めるようにするためと、実行が数千件を求める場所で、テストが必要な数件だけを
// 求められるようにするため。
type amounts struct {
	// mainPosts is how many everyday posts roleMain holds, spread over the
	// three years it has been here. This is the count the export is checked
	// against: the archive holds one HTML file per month, so three years of
	// posts is what makes those files line up in the zip, and what gives the
	// table of contents a count beside each month that differs from the next.
	//
	// [Ja] mainPosts は、roleMain が持つ日常ポストの件数。ここにいる 3 年間へ
	// 散らして配置する。エクスポートの確認対象となるのがこの件数である。アーカイブは
	// 月ごとに 1 つの HTML ファイルを持つため、3 年分のポストがあることで、それらの
	// ファイルが zip の中に並び、目次には月ごとに異なる件数が並ぶ。
	mainPosts int

	// followerPosts, englishPosts and discardedPosts are how many posts the
	// remaining roles hold. They are counted in tens rather than thousands:
	// the export is checked against roleMain alone, and what these three are
	// there to show — a home timeline with somebody else's posts in it, an
	// archive whose months break at a different hour, and posts whose author
	// is gone — needs a screenful each rather than a history.
	//
	// [Ja] followerPosts・englishPosts・discardedPosts は、残りの役割が持つ
	// ポストの件数。数千件ではなく数十件で数える。エクスポートの確認対象は roleMain
	// だけであり、この 3 つが示すためにいるもの (他人のポストが並ぶホームタイム
	// ライン、月が別の時刻で切り替わるアーカイブ、作者が居なくなったポスト) には、
	// 履歴ではなく 1 画面分があれば足りるため。
	followerPosts  int
	englishPosts   int
	discardedPosts int

	// followerToMainStamps, mainToFollowerStamps and newcomerToMainStamps are
	// how many stamps each direction puts on the target's posts. They are
	// named by both ends of the direction because roleMain's posts are stamped
	// from two of them, and a count named after whose posts it lands on would
	// no longer say which direction it belongs to.
	//
	// The two directions between roleMain and roleFollower differ in size so
	// that the notification list can be read once with more than a page of
	// entries in it and once with a handful: a run that gave both ends the
	// same number would not show that what the list displays is the
	// recipient's own.
	//
	// [Ja] followerToMainStamps・mainToFollowerStamps・newcomerToMainStamps は、
	// 各向きが相手のポストへ何件のスタンプを押すか。向きの両端で名前を付けているのは、
	// roleMain のポストへは 2 つの向きからスタンプが押されるためである。どのポストへ
	// 押すのかで件数を名付けると、それがどの向きのものなのかを言えなくなる。
	//
	// roleMain と roleFollower の間の 2 つの向きは件数を違える。通知一覧を、1 ページを
	// 超える件数がある状態と数件しかない状態の双方で読めるようにするためである。両端へ
	// 同じ件数を与える実行では、一覧が表示しているものが受け取り手自身のものであることを
	// 確認できない。
	followerToMainStamps int
	mainToFollowerStamps int
	newcomerToMainStamps int
}

// defaultAmounts is what a run creates. Tests pass amounts of their own.
//
// [Ja] defaultAmounts は実行 1 回分が作る件数。テストは自前の件数を渡す。
var defaultAmounts = amounts{
	mainPosts:            3000,
	followerPosts:        48,
	englishPosts:         16,
	discardedPosts:       5,
	followerToMainStamps: 40,
	mainToFollowerStamps: 6,
	newcomerToMainStamps: 4,
}

// followerPostMonths is how far back roleFollower's posts reach. Half a year
// is short against the thirty months it has been here, on purpose: its posts
// are read in a home timeline and under a followed profile, both of which show
// the newest first, and a few dozen posts spread over its whole membership
// would leave those screens with about one post a month.
//
// [Ja] followerPostMonths は、roleFollower のポストがどこまで遡るか。ここにいる
// 30 か月に対して半年は短いが、これは意図的である。この役割のポストが読まれるのは
// ホームタイムラインとフォロー中のプロフィールで、どちらも新しいものから表示する。
// 数十件を在籍期間の全体へ散らすと、それらの画面には月に 1 件ほどしか並ばなくなる。
const followerPostMonths = 6

// englishPostMonths is how far back roleEnglish's posts reach. Three months is
// what an English archive needs to be worth opening: the month boundary is
// what this role is there to show against roleMain's, and a boundary can only
// be looked at between two months that both hold posts.
//
// [Ja] englishPostMonths は、roleEnglish のポストがどこまで遡るか。英語の
// アーカイブを開く意味があるのは 3 か月分からになる。この役割が roleMain と並べて
// 示すのは月境界であり、境界は、双方ともポストを持つ 2 つの月の間でしか見られない。
const englishPostMonths = 3

// discardedPostMonths is how far back roleDiscarded's posts reach. What this
// role shows is how a post looks once its author is gone, which is visible in
// a single post, so its posts only need to be recent enough to be reached.
//
// [Ja] discardedPostMonths は、roleDiscarded のポストがどこまで遡るか。この役割が
// 示すのは、作者が居なくなったポストの見え方であり、それは 1 件でも見える。ポストは
// 辿り着ける程度に新しければ足りる。
const discardedPostMonths = 2

// monthlyPostWeights is how a role's posts are shared out between the months
// they span, one weight per month, cycled from the oldest month onwards.
//
// The months are deliberately given different shares. If every month held the
// same number of posts, the counts the export's table of contents shows could
// not be told apart from a count that was taken once and repeated, which is
// the mistake those numbers are looked at to catch.
//
// The list is seventeen long so that it does not line up with the calendar. A
// list of twelve would give the same month of every year the same count, and a
// table of contents where March is 64 posts in each of three years reads as
// generated rather than as a history.
//
// [Ja] monthlyPostWeights は、ある役割のポストが、それが広がる月々へどう配分される
// かを月ごとに 1 つの重みで表したもの。最も古い月から順に、繰り返し使う。
//
// 月ごとの取り分を意図的に変えている。どの月も同じ件数であれば、エクスポートの目次が
// 見せる件数は、1 度数えたものを繰り返しているだけの状態と区別できない。それこそが、
// あの数字を見て捕まえたい誤りである。
//
// 一覧を 17 個にしているのは、暦と重ならないようにするため。12 個であれば毎年の同じ
// 月が同じ件数になり、3 年分の 3 月がいずれも 64 件という目次は、履歴ではなく生成物
// として読まれることになる。
var monthlyPostWeights = []int{5, 2, 4, 3, 6, 2, 5, 3, 4, 2, 6, 3, 5, 2, 4, 6, 3}

// mainPostBodies are what roleMain's everyday posts are written from, cycled
// through in order. They are dull on purpose: these posts exist to be counted
// by a month's listing and to fill an archive, and anything worth reading here
// would only compete with the posts that exist to be read (the ones written
// for the export's output contract).
//
// The wording follows what the profile says about itself, so that an account
// that describes itself as writing about coffee and walks is not one whose
// posts are about something else.
//
// [Ja] mainPostBodies は roleMain の日常ポストを書き起こす材料で、順に繰り返し
// 使う。意図的に退屈な内容にしている。これらのポストは月ごとの一覧に数えられ、
// アーカイブを埋めるために存在しており、ここに読ませたい内容を書いても、読ませる
// ために存在するポスト (エクスポートの出力契約のために書いたもの) と競合するだけで
// あるため。
//
// 文面はプロフィールの自己紹介に沿わせている。コーヒーと散歩について書くと自称して
// いるアカウントが、別の話題ばかり書いている状態にならないようにするため。
var mainPostBodies = []string{
	"朝いちばんに豆を挽いた。今日は少し粗めにしてみる。",
	"川沿いを 30 分ほど歩いた。この時間はまだ人が少ない。",
	"写真を撮ろうとしたら電池が切れていた。こういう日もある。",
	"新しい豆を買ってみた。浅煎りを選ぶのは久しぶり。",
	"夕方の光がよかったので、遠回りして帰った。",
	"散歩の途中で喫茶店を見つけた。今度は中に入ってみたい。",
	"雨。傘を差して歩くのも、たまには悪くない。",
	"現像したら思っていたより暗かった。次はもう少し明るめに撮る。",
	"同じ道でも、季節が変わると別の道に見える。",
	"淹れ方を変えてみたら、味がずいぶん違った。",
	"歩数計を見たら、今日はよく歩いていた。",
	"カメラを持たずに出た日にかぎって、いい景色に出会う。",
	"豆が切れた。明日は買いに行かないと。",
	"公園のベンチでしばらくぼんやりしていた。",
	"朝の散歩を習慣にしてから、夜よく眠れる。",
	"曇りの日の写真は、色が落ち着いていて好き。",
	"コーヒーを淹れながら、今日やることを書き出した。",
	"遠くの山がきれいに見えた。空気が澄んでいる。",
}

// followerPostBodies are what roleFollower writes. It has a list of its own
// rather than sharing roleMain's, because the two accounts' posts are read
// side by side in a home timeline, and a timeline where both of them wrote the
// same sentences would not look like two people.
//
// [Ja] followerPostBodies は roleFollower が書く内容。roleMain と共有せず自前の
// 一覧を持つのは、2 つのアカウントのポストがホームタイムラインで並べて読まれるため。
// どちらも同じ文を書いているタイムラインは、2 人の人物には見えない。
var followerPostBodies = []string{
	"近所のパン屋、今日はカンパーニュが焼き上がっていた。",
	"サウナのあとの外気浴がいちばん気持ちいい。",
	"食パンを買いすぎたので、明日はトーストばかりになりそう。",
	"初めての店に入ってみた。クロワッサンがよかった。",
	"水風呂が冷たすぎて、10 秒で出てしまった。",
	"夕方に行くとパンが安くなることを知ってしまった。",
	"サウナ室で本を読んでいる人がいた。熱くないのだろうか。",
	"あんぱんとカレーパンで迷って、結局どちらも買った。",
	"今日は空いていて、ととのうまでに時間がかからなかった。",
	"新しいパン屋ができるらしい。開店が楽しみ。",
}

// englishPostBodies are what roleEnglish writes. They stay in English because
// this role is the one an English archive is read from, and an archive whose
// wording is English while its posts are Japanese would only show half of what
// the locale decides.
//
// [Ja] englishPostBodies は roleEnglish が書く内容。英語のままにしているのは、
// この役割が英語のアーカイブを読むためのものであるため。文言が英語でポストが日本語の
// アーカイブでは、ロケールが決めているものの半分しか見えない。
var englishPostBodies = []string{
	"Rain all morning. I spent ten minutes deciding whether to take an umbrella.",
	"Walked by the river after work. It is quietest at this hour.",
	"Tried the coffee place near the station. The window seats are the good ones.",
	"Finished a book that had been sitting on the shelf since spring.",
	"Cooked dinner out of what was left in the fridge. Better than expected.",
	"The wind is strong today. Cycling into it feels like standing still.",
	"Making tea before bed seems to have become a habit.",
	"The construction by the station is finally done. The pavement is wider now.",
	"Sat at my desk all day. Tomorrow I would like to go somewhere.",
	"Something is sprouting in the pot on the balcony. The watering paid off.",
}

// discardedPostBodies are what roleDiscarded wrote before its profile was
// deleted. The list is short because these posts are looked at one at a time:
// what they show is how a post reads once the account behind it is gone.
//
// [Ja] discardedPostBodies は、roleDiscarded がプロフィールを削除される前に書いた
// 内容。一覧が短いのは、これらのポストが 1 件ずつ見られるものであるため。示すのは、
// その背後のアカウントが居なくなったポストがどう読めるのかということである。
var discardedPostBodies = []string{
	"先週の山で撮った写真を整理している。",
	"登り始めは霧だったのに、頂上では晴れていた。",
	"レンズを 1 本だけ持っていくと決めると、荷物が軽くなる。",
	"次はもう少し早い時間に登りたい。",
}

// rolePostSpec is one role's share of the everyday posts: how many it holds,
// how far back they reach, and what they are written from.
//
// [Ja] rolePostSpec は、日常ポストのうちある役割が持つ分。件数・どこまで遡るか・
// 何から書き起こすかを持つ。
type rolePostSpec struct {
	role   seedRole
	count  int
	months int
	bodies []string
}

// rolePostSpecs is what each role holds, with the counts taken from amt.
//
// roleNewcomer is absent, which is what gives it no posts. An account that can
// reach every screen while having written nothing is the only way to see an
// empty export and a profile page with nothing under it, so its absence here
// is the whole of what that role is.
//
// [Ja] rolePostSpecs は各役割が持つ分。件数は amt から取る。
//
// roleNewcomer は含まれておらず、それがこの役割にポストを持たせない方法になる。
// すべての画面へ辿り着けるのに何も書いていないアカウントは、空のエクスポートと、
// 下に何も並ばないプロフィール画面を見るための唯一の手段であり、ここに無いことが
// その役割のすべてである。
func rolePostSpecs(amt amounts) []rolePostSpec {
	return []rolePostSpec{
		{role: roleMain, count: amt.mainPosts, months: historyMonths, bodies: mainPostBodies},
		{role: roleFollower, count: amt.followerPosts, months: followerPostMonths, bodies: followerPostBodies},
		{role: roleEnglish, count: amt.englishPosts, months: englishPostMonths, bodies: englishPostBodies},
		{role: roleDiscarded, count: amt.discardedPosts, months: discardedPostMonths, bodies: discardedPostBodies},
	}
}

// postWriter writes the posts of one run.
//
// The application every post is attributed to and the instant the run is
// anchored to are held here rather than passed down each call, because both
// are fixed for the whole run: a post that named a different application, or
// a run that read the clock again partway through, would be describing a
// different run than the one that started.
//
// [Ja] postWriter は、実行 1 回分のポストを書き込む。
//
// すべてのポストの帰属先となるアプリケーションと、実行が基準とする時点を、呼び出し
// ごとに渡さずここで持つ。どちらも実行全体で固定であるため。別のアプリケーションを
// 名指しするポストや、途中でもう一度時計を読む実行は、始まったときの実行とは別の
// 実行を記述することになる。
type postWriter struct {
	tx            *sql.Tx
	posts         *repository.PostRepository
	profiles      *repository.ProfileRepository
	applicationID model.OauthApplicationID
	now           time.Time
}

// newPostWriter binds a post writer to the run's transaction, to the
// application its posts are attributed to, and to the instant the run is
// anchored to.
//
// It is a function rather than a literal at the one place posts are written,
// because posts are written from more than one generator: the everyday posts
// and the posts that carry a link card are both posts, and a second literal
// would be a second place for the repositories they are written through to
// drift apart.
//
// [Ja] newPostWriter は、ポストの書き込み手を、実行のトランザクション・ポストの
// 帰属先となるアプリケーション・実行が基準とする時点へ束ねる。
//
// ポストを書き込む 1 箇所へリテラルで置かず関数にしているのは、ポストを書き込む
// 生成器が 1 つではないため。日常ポストも、リンクカードを持つポストも、どちらも
// ポストであり、2 つ目のリテラルは、それらが書き込まれる先のリポジトリが食い違う
// 2 つ目の場所になる。
func newPostWriter(tx *sql.Tx, applicationID model.OauthApplicationID, now time.Time) *postWriter {
	q := query.New(tx)

	return &postWriter{
		tx:            tx,
		posts:         repository.NewPostRepository(q),
		profiles:      repository.NewProfileRepository(q),
		applicationID: applicationID,
		now:           now,
	}
}

// createPosts writes the everyday posts of every role that holds them.
//
// [Ja] createPosts は、日常ポストを持つすべての役割について、そのポストを書き込む。
func createPosts(
	ctx context.Context,
	tx *sql.Tx,
	amt amounts,
	applicationID model.OauthApplicationID,
	accounts []seedAccount,
	now time.Time,
) error {
	writer := newPostWriter(tx, applicationID, now)

	for _, spec := range rolePostSpecs(amt) {
		account, err := accountForRole(accounts, spec.role)
		if err != nil {
			return err
		}

		if err := writer.writeRolePosts(ctx, account, spec); err != nil {
			return fmt.Errorf("役割 %s のポストの作成に失敗: %w", spec.role, err)
		}
	}

	// The posts written for the export's output contract go to roleMain
	// alone, and they go after its everyday posts: they are placed in months
	// the everyday posts have already been laid across, and the profile's
	// newest post should be one of the everyday ones.
	//
	// [Ja] エクスポートの出力契約のために書くポストは roleMain だけに置き、その
	// 日常ポストの後に書く。これらは日常ポストがすでに敷かれた月々へ配置されるもので
	// あり、プロフィールの最も新しいポストは日常ポストのひとつであるべきであるため。
	main, err := accountForRole(accounts, roleMain)
	if err != nil {
		return err
	}

	if err := writer.writeVariationPosts(ctx, main); err != nil {
		return fmt.Errorf("役割 %s のエクスポート確認用ポストの作成に失敗: %w", roleMain, err)
	}

	return nil
}

// writeRolePosts writes one role's posts, oldest first, and records the newest
// of them on the profile.
//
// The months are counted in the account's own time zone. Which month a post
// falls into is decided by the reader's clock, and the export splits its
// archive on that boundary, so a run that counted the months in the time zone
// of whichever machine it happened to run on would place the same post in
// different files depending on where it was seeded.
//
// [Ja] writeRolePosts は、ある役割のポストを古いものから書き込み、その中で最も
// 新しいものをプロフィールへ記録する。
//
// 月はアカウント自身のタイムゾーンで数える。ポストがどの月に入るのかは読み手の時計が
// 決めるものであり、エクスポートはその境界でアーカイブを分割する。実行したマシンの
// タイムゾーンで月を数える実行は、どこでシードしたかによって同じポストを別のファイル
// へ置くことになる。
func (w *postWriter) writeRolePosts(ctx context.Context, account seedAccount, spec rolePostSpec) error {
	// A spec with no months has nowhere to place its posts. Returning here
	// rather than falling through is what leaves last_post_at unset: a
	// profile whose newest post is the zero instant reads as one that posted
	// in the year one.
	//
	// [Ja] 月を持たない spec には、ポストを置く先が無い。そのまま先へ進まずここで
	// 戻ることが、last_post_at を未設定のままにする。最も新しいポストがゼロ値の
	// 時点であるプロフィールは、西暦 1 年にポストしたものとして読まれる。
	if spec.count <= 0 || spec.months <= 0 {
		return nil
	}

	// The zone is read from the account rather than resolved again from the
	// roster: the roster checked at load time that it names a real zone, and
	// what the screens use is the value that reached the user row.
	//
	// [Ja] タイムゾーンは名簿から引き直さずアカウントから読む。名簿は読み込み時に
	// それが実在するタイムゾーンを指していることを検査しており、画面が使うのは
	// ユーザー行へ届いた値であるため。
	location, err := time.LoadLocation(account.user.TimeZone)
	if err != nil {
		return fmt.Errorf("タイムゾーン %q の読み込みに失敗: %w", account.user.TimeZone, err)
	}

	var lastPostAt time.Time

	body := 0
	for month, count := range monthlyPostCounts(spec.count, spec.months) {
		start, end := monthWindow(w.now, location, spec.months, month)

		for i := range count {
			publishedAt := storedInstant(postTimeInWindow(start, end, i, count))

			if _, err := w.writePost(ctx, account, spec.bodies[body%len(spec.bodies)], publishedAt); err != nil {
				return err
			}

			body++
			lastPostAt = publishedAt
		}
	}

	// last_post_at is what the screens that order profiles by activity read,
	// so it is set from the posts that were just written rather than left for
	// the next post the account makes through the application.
	//
	// [Ja] last_post_at は、プロフィールを活動順に並べる画面が読む値であるため、
	// アカウントがアプリケーションを通して次にポストするときまで待たず、たった今
	// 書き込んだポストから設定する。
	return w.recordLastPostAt(ctx, account, lastPostAt)
}

// recordLastPostAt records at as the profile's newest post, unless the profile
// already holds a newer one.
//
// The comparison lives here rather than at either call site because a
// profile's posts are written in more than one pass: the everyday posts first,
// then the posts written for the export's output contract. Which pass holds
// the newest post is a property of where those posts were placed, not of the
// order the passes run in.
//
// [Ja] recordLastPostAt は、プロフィールがすでにより新しいものを持っている場合を
// 除いて、at をそのプロフィールの最も新しいポストとして記録する。
//
// 比較を呼び出し側ではなくここに置くのは、プロフィールのポストが複数の段階に
// 分かれて書き込まれるため。まず日常ポスト、次にエクスポートの出力契約のために書く
// ポストとなる。どちらの段階が最も新しいポストを持つのかは、それらのポストがどこへ
// 配置されたかによって決まるものであり、段階の実行順によって決まるものではない。
func (w *postWriter) recordLastPostAt(ctx context.Context, account seedAccount, at time.Time) error {
	if account.profile.LastPostAt != nil && !at.After(*account.profile.LastPostAt) {
		return nil
	}

	if err := w.profiles.UpdateLastPostAt(ctx, account.profile.ID, at); err != nil {
		return fmt.Errorf("プロフィールの last_post_at の更新に失敗: %w", err)
	}

	// The model is brought along with the row it stands for, so that a
	// generator that receives this account afterwards is not handed a profile
	// that says nothing has been written under it.
	//
	// [Ja] モデルは、それが表す行と一緒に更新する。この後にこのアカウントを受け取る
	// 生成器へ、まだ何も書かれていないと告げるプロフィールを渡さないようにするため。
	account.profile.LastPostAt = &at

	return nil
}

// writePost writes one post with an ID ordered by its stored publication time,
// and returns that ID.
//
// The ID is returned rather than looked up again by whoever needs it: a post
// is found by its profile and its publication time, and two posts of the same
// account can share an instant, so a lookup would be answering a question the
// caller already knows the answer to and could get wrong.
//
// [Ja] writePost は、保存された公開日時順に並ぶIDで 1 件のポストを書き込み、その
// IDを返す。
//
// IDを必要とする側が引き直すのではなく戻り値で渡す。ポストはプロフィールと公開日時で
// 引くことになるが、同じアカウントの 2 件のポストは同じ時点を共有しうるため、引き直しは、
// 呼び出し側がすでに答えを持っている問いに、誤りうる形で答えることになる。
func (w *postWriter) writePost(
	ctx context.Context,
	account seedAccount,
	body string,
	publishedAt time.Time,
) (model.PostID, error) {
	post, err := w.posts.Create(ctx, repository.CreatePostInput{
		ProfileID:          account.profile.ID,
		Content:            body,
		PublishedAt:        publishedAt,
		OauthApplicationID: w.applicationID,
	})
	if err != nil {
		return model.PostID{}, fmt.Errorf("ポストの作成に失敗: %w", err)
	}

	// Rails' PostRecord.prev_post / next_post filter by ID. The ULID's first
	// 48 bits must therefore use publication milliseconds rather than the
	// insertion clock. The next 10 bits hold the microseconds within that
	// millisecond, preserving the column's full ordering even when posts share
	// a millisecond. The remaining 70 random bits keep IDs distinct, including
	// posts with identical publication times.
	//
	// Adjust the ID in the seed transaction before any dependent rows exist.
	//
	// [Ja] Railsの PostRecord.prev_post / next_post はIDで絞り込むため、ULIDの
	// 先頭48ビットには挿入時刻ではなく公開日時のミリ秒を設定する。続く10ビットに
	// ミリ秒内のマイクロ秒を格納し、同じミリ秒のポストでもカラムの精度で順序を保つ。
	// 残り70ビットの乱数で、公開日時が同じポストも含めてIDを区別する。
	//
	// 関連行を作成する前に、シードのトランザクション内でIDを補正する。
	id := uuid.UUID(post.ID)
	milliseconds := post.PublishedAt.UnixMilli()
	for i := 5; i >= 0; i-- {
		id[i] = byte(milliseconds & 0xff)
		milliseconds >>= 8
	}
	microseconds := post.PublishedAt.Nanosecond() / 1000 % 1000
	id[6] = byte((microseconds >> 2) & 0xff)
	id[7] = byte(microseconds&3)<<6 | id[7]&0x3f

	if _, err := w.tx.ExecContext(ctx, `
		UPDATE posts SET id = $2 WHERE id = $1
	`, uuid.UUID(post.ID), id); err != nil {
		return model.PostID{}, fmt.Errorf("ポストIDの公開日時への補正に失敗: %w", err)
	}

	return model.PostID(id), nil
}

// storedInstant is how an instant is handed to the database.
//
// It is offered in UTC. posts.published_at and profiles.last_post_at are
// timestamps without a time zone, and PostgreSQL takes the wall clock of what
// it is given and drops the offset, so an instant offered in another zone
// would be stored as that zone's digits and read back as an instant hours away
// from the one that was meant — on the far side of a month boundary, for a
// post placed near the end of a month.
//
// It is rounded here rather than left to the column, so that the models the
// generators carry on to each other describe the rows that were written. A
// column that holds microseconds rounds a nanosecond away from what the value
// in hand still says.
//
// [Ja] storedInstant は、時点をデータベースへ渡すときの形。
//
// 時点は UTC で渡す。posts.published_at と profiles.last_post_at はタイムゾーンを
// 持たない timestamp であり、PostgreSQL は与えられた値の壁時計を取ってオフセットを
// 捨てる。別のタイムゾーンで渡した時点は、そのタイムゾーンの数字として保存され、
// 後から読むと意図した時点から数時間ずれた時点になる。月の終わり近くに置いたポストで
// あれば、月境界の向こう側へ移ることになる。
//
// 丸めをカラムに任せずここで行うのは、生成器が互いに渡し合うモデルが、書き込まれた行を
// 記述しているようにするため。マイクロ秒を保持するカラムはナノ秒を丸めるが、手元の値は
// それを持ったままになる。
func storedInstant(at time.Time) time.Time {
	return at.UTC().Truncate(time.Microsecond)
}

// monthlyPostCounts shares total posts out between months months, oldest
// first, in the proportions monthlyPostWeights gives them.
//
// The remainder that integer division leaves over is handed to the newest
// months. A run is looked at from the newest post backwards, so the months a
// developer opens first are the ones that should not be the short ones.
//
// [Ja] monthlyPostCounts は、合計 total 件のポストを、古い月から順に months 個の月へ、
// monthlyPostWeights が与える比率で配分する。
//
// 整数除算が残した端数は新しい月へ渡す。実行は最も新しいポストから遡って見られる
// ため、開発者が最初に開く月が、端数で少なくなっている月であってはならない。
func monthlyPostCounts(total, months int) []int {
	if months <= 0 {
		return nil
	}

	counts := make([]int, months)
	if total <= 0 {
		return counts
	}

	totalWeight := 0
	for month := range months {
		totalWeight += monthlyPostWeights[month%len(monthlyPostWeights)]
	}

	assigned := 0
	for month := range months {
		counts[month] = total * monthlyPostWeights[month%len(monthlyPostWeights)] / totalWeight
		assigned += counts[month]
	}

	for month := months - 1; assigned < total; month = (month + months - 1) % months {
		counts[month]++
		assigned++
	}

	return counts
}

// monthWindow returns the half-open interval the month at index covers, where
// index 0 is the oldest of months months and months-1 is the month the run is
// happening in.
//
// The last window is cut short at the run's own instant. A post is something
// that was written, so a month still in progress reaches up to now and no
// further: the alternative is an account whose newest post has not happened
// yet.
//
// [Ja] monthWindow は、index 番目の月が覆う半開区間を返す。index 0 は months 個の
// 月のうち最も古い月であり、months-1 は実行が行われている月である。
//
// 最後の区間は実行自身の時点で打ち切る。ポストは書かれたものであるため、進行中の月は
// 現在までで、その先へは伸びない。そうしなければ、最も新しいポストがまだ起きていない
// アカウントができてしまう。
func monthWindow(now time.Time, location *time.Location, months, index int) (time.Time, time.Time) {
	local := now.In(location)
	currentMonthStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)

	start := currentMonthStart.AddDate(0, index-(months-1), 0)
	end := start.AddDate(0, 1, 0)
	if end.After(now) {
		end = now
	}

	return start, end
}

// postTimeInWindow places the index-th of count posts inside a month, spacing
// them evenly and leaving a gap at either end of the month.
//
// The posts of a month are spread rather than stamped with one instant,
// because a month whose posts all share a timestamp is a month whose order is
// decided by whatever the database returns first.
//
// [Ja] postTimeInWindow は、ある月の count 件のうち index 番目のポストを、等間隔に、
// 月の両端に余白を残して配置する。
//
// 月のポストを 1 つの時点で押さずに散らすのは、すべてのポストが同じ時刻を持つ月では、
// その並び順をデータベースが先に返したものが決めることになるため。
func postTimeInWindow(start, end time.Time, index, count int) time.Time {
	// The interval is taken before it is multiplied, not after. A 31-day
	// month is 2.68e15 nanoseconds and an int64 Duration reaches 9.22e18, so
	// the multiply-first form overflows once the count passes some 3,400.
	// What reaches here is one month's share — about 130 posts at the volume
	// a run creates — so neither order overflows today, and dividing first is
	// what keeps that true however the counts in amounts are raised.
	//
	// [Ja] 間隔は掛ける前に割る。31 日の月は 2.68e15 ナノ秒で、Duration の実体で
	// ある int64 が届くのは 9.22e18 までであるため、先に掛ける形は件数が 3,400 ほど
	// を超えたところであふれる。ここへ渡るのは 1 か月分の取り分 (実行 1 回分の生成量
	// で 130 件ほど) であり、今はどちらの順序でもあふれないが、割ってから掛ける形は、
	// amounts の件数をどう増やしてもそれが成り立ち続けるようにする。
	interval := end.Sub(start) / time.Duration(count+1)

	return start.Add(interval * time.Duration(index+1))
}
