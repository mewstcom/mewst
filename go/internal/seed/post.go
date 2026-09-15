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

// amountsは、実行1回分が各種の行を何件作るか。件数を、それを読むループの
// かたわらではなく1つの構造体にまとめているのは、実行が何を作るのかを1箇所で
// 読めるようにするためと、実行が数千件を求める場所で、テストが必要な数件だけを
// 求められるようにするため。
type amounts struct {
	// mainPostsは、roleMainが持つ日常ポストの件数。ここにいる3年間へ
	// 散らして配置する。エクスポートの確認対象となるのがこの件数である。アーカイブは
	// 月ごとに1つのHTMLファイルを持つため、3年分のポストがあることで、それらの
	// ファイルがzipの中に並び、目次には月ごとに異なる件数が並ぶ。
	mainPosts int

	// followerPosts・englishPosts・discardedPostsは、残りの役割が持つ
	// ポストの件数。数千件ではなく数十件で数える。エクスポートの確認対象はroleMain
	// だけであり、この3つが示すためにいるもの (他人のポストが並ぶホームタイム
	// ライン、月が別の時刻で切り替わるアーカイブ、作者が居なくなったポスト) には、
	// 履歴ではなく1画面分があれば足りるため。
	followerPosts  int
	englishPosts   int
	discardedPosts int

	// followerToMainStamps・mainToFollowerStamps・newcomerToMainStampsは、
	// 各向きが相手のポストへ何件のスタンプを押すか。向きの両端で名前を付けているのは、
	// roleMainのポストへは2つの向きからスタンプが押されるためである。どのポストへ
	// 押すのかで件数を名付けると、それがどの向きのものなのかを言えなくなる。
	//
	// roleMainとroleFollowerの間の2つの向きは件数を違える。通知一覧を、1ページを
	// 超える件数がある状態と数件しかない状態の双方で読めるようにするためである。両端へ
	// 同じ件数を与える実行では、一覧が表示しているものが受け取り手自身のものであることを
	// 確認できない。
	followerToMainStamps int
	mainToFollowerStamps int
	newcomerToMainStamps int
}

// defaultAmountsは実行1回分が作る件数。テストは自前の件数を渡す。
var defaultAmounts = amounts{
	mainPosts:            3000,
	followerPosts:        48,
	englishPosts:         16,
	discardedPosts:       5,
	followerToMainStamps: 40,
	mainToFollowerStamps: 6,
	newcomerToMainStamps: 4,
}

// followerPostMonthsは、roleFollowerのポストがどこまで遡るか。ここにいる
// 30か月に対して半年は短いが、これは意図的である。この役割のポストが読まれるのは
// ホームタイムラインとフォロー中のプロフィールで、どちらも新しいものから表示する。
// 数十件を在籍期間の全体へ散らすと、それらの画面には月に1件ほどしか並ばなくなる。
const followerPostMonths = 6

// englishPostMonthsは、roleEnglishのポストがどこまで遡るか。英語の
// アーカイブを開く意味があるのは3か月分からになる。この役割がroleMainと並べて
// 示すのは月境界であり、境界は、双方ともポストを持つ2つの月の間でしか見られない。
const englishPostMonths = 3

// discardedPostMonthsは、roleDiscardedのポストがどこまで遡るか。この役割が
// 示すのは、作者が居なくなったポストの見え方であり、それは1件でも見える。ポストは
// 辿り着ける程度に新しければ足りる。
const discardedPostMonths = 2

// monthlyPostWeightsは、ある役割のポストが、それが広がる月々へどう配分される
// かを月ごとに1つの重みで表したもの。最も古い月から順に、繰り返し使う。
//
// 月ごとの取り分を意図的に変えている。どの月も同じ件数であれば、エクスポートの目次が
// 見せる件数は、1度数えたものを繰り返しているだけの状態と区別できない。それこそが、
// あの数字を見て捕まえたい誤りである。
//
// 一覧を17個にしているのは、暦と重ならないようにするため。12個であれば毎年の同じ
// 月が同じ件数になり、3年分の3月がいずれも64件という目次は、履歴ではなく生成物
// として読まれることになる。
var monthlyPostWeights = []int{5, 2, 4, 3, 6, 2, 5, 3, 4, 2, 6, 3, 5, 2, 4, 6, 3}

// mainPostBodiesはroleMainの日常ポストを書き起こす材料で、順に繰り返し
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

// followerPostBodiesはroleFollowerが書く内容。roleMainと共有せず自前の
// 一覧を持つのは、2つのアカウントのポストがホームタイムラインで並べて読まれるため。
// どちらも同じ文を書いているタイムラインは、2人の人物には見えない。
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

// englishPostBodiesはroleEnglishが書く内容。英語のままにしているのは、
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

// discardedPostBodiesは、roleDiscardedがプロフィールを削除される前に書いた
// 内容。一覧が短いのは、これらのポストが1件ずつ見られるものであるため。示すのは、
// その背後のアカウントが居なくなったポストがどう読めるのかということである。
var discardedPostBodies = []string{
	"先週の山で撮った写真を整理している。",
	"登り始めは霧だったのに、頂上では晴れていた。",
	"レンズを 1 本だけ持っていくと決めると、荷物が軽くなる。",
	"次はもう少し早い時間に登りたい。",
}

// rolePostSpecは、日常ポストのうちある役割が持つ分。件数・どこまで遡るか・
// 何から書き起こすかを持つ。
type rolePostSpec struct {
	role   seedRole
	count  int
	months int
	bodies []string
}

// rolePostSpecsは各役割が持つ分。件数はamtから取る。
//
// roleNewcomerは含まれておらず、それがこの役割にポストを持たせない方法になる。
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

// postWriterは、実行1回分のポストを書き込む。
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

// newPostWriterは、ポストの書き込み手を、実行のトランザクション・ポストの
// 帰属先となるアプリケーション・実行が基準とする時点へ束ねる。
//
// ポストを書き込む1箇所へリテラルで置かず関数にしているのは、ポストを書き込む
// 生成器が1つではないため。日常ポストも、リンクカードを持つポストも、どちらも
// ポストであり、2つ目のリテラルは、それらが書き込まれる先のリポジトリが食い違う
// 2つ目の場所になる。
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

// createPostsは、日常ポストを持つすべての役割について、そのポストを書き込む。
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
			return fmt.Errorf("役割%sのポストの作成に失敗: %w", spec.role, err)
		}
	}

	// エクスポートの出力契約のために書くポストはroleMainだけに置き、その
	// 日常ポストの後に書く。これらは日常ポストがすでに敷かれた月々へ配置されるもので
	// あり、プロフィールの最も新しいポストは日常ポストのひとつであるべきであるため。
	main, err := accountForRole(accounts, roleMain)
	if err != nil {
		return err
	}

	if err := writer.writeVariationPosts(ctx, main); err != nil {
		return fmt.Errorf("役割%sのエクスポート確認用ポストの作成に失敗: %w", roleMain, err)
	}

	return nil
}

// writeRolePostsは、ある役割のポストを古いものから書き込み、その中で最も
// 新しいものをプロフィールへ記録する。
//
// 月はアカウント自身のタイムゾーンで数える。ポストがどの月に入るのかは読み手の時計が
// 決めるものであり、エクスポートはその境界でアーカイブを分割する。実行したマシンの
// タイムゾーンで月を数える実行は、どこでシードしたかによって同じポストを別のファイル
// へ置くことになる。
func (w *postWriter) writeRolePosts(ctx context.Context, account seedAccount, spec rolePostSpec) error {
	// 月を持たないspecには、ポストを置く先が無い。そのまま先へ進まずここで
	// 戻ることが、last_post_atを未設定のままにする。最も新しいポストがゼロ値の
	// 時点であるプロフィールは、西暦1年にポストしたものとして読まれる。
	if spec.count <= 0 || spec.months <= 0 {
		return nil
	}

	// タイムゾーンは名簿から引き直さずアカウントから読む。名簿は読み込み時に
	// それが実在するタイムゾーンを指していることを検査しており、画面が使うのは
	// ユーザー行へ届いた値であるため。
	location, err := time.LoadLocation(account.user.TimeZone)
	if err != nil {
		return fmt.Errorf("タイムゾーン%qの読み込みに失敗: %w", account.user.TimeZone, err)
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

	// last_post_atは、プロフィールを活動順に並べる画面が読む値であるため、
	// アカウントがアプリケーションを通して次にポストするときまで待たず、たった今
	// 書き込んだポストから設定する。
	return w.recordLastPostAt(ctx, account, lastPostAt)
}

// recordLastPostAtは、プロフィールがすでにより新しいものを持っている場合を
// 除いて、atをそのプロフィールの最も新しいポストとして記録する。
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
		return fmt.Errorf("プロフィールのlast_post_atの更新に失敗: %w", err)
	}

	// モデルは、それが表す行と一緒に更新する。この後にこのアカウントを受け取る
	// 生成器へ、まだ何も書かれていないと告げるプロフィールを渡さないようにするため。
	account.profile.LastPostAt = &at

	return nil
}

// writePostは、保存された公開日時順に並ぶIDで1件のポストを書き込み、その
// IDを返す。
//
// IDを必要とする側が引き直すのではなく戻り値で渡す。ポストはプロフィールと公開日時で
// 引くことになるが、同じアカウントの2件のポストは同じ時点を共有しうるため、引き直しは、
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

	// RailsのPostRecord.prev_post / next_postはIDで絞り込むため、ULIDの
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

// storedInstantは、時点をデータベースへ渡すときの形。
//
// 時点はUTCで渡す。posts.published_atとprofiles.last_post_atはタイムゾーンを
// 持たないtimestampであり、PostgreSQLは与えられた値の壁時計を取ってオフセットを
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

// monthlyPostCountsは、合計total件のポストを、古い月から順にmonths個の月へ、
// monthlyPostWeightsが与える比率で配分する。
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

// monthWindowは、index番目の月が覆う半開区間を返す。index 0はmonths個の
// 月のうち最も古い月であり、months-1は実行が行われている月である。
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

// postTimeInWindowは、ある月のcount件のうちindex番目のポストを、等間隔に、
// 月の両端に余白を残して配置する。
//
// 月のポストを1つの時点で押さずに散らすのは、すべてのポストが同じ時刻を持つ月では、
// その並び順をデータベースが先に返したものが決めることになるため。
func postTimeInWindow(start, end time.Time, index, count int) time.Time {
	// 間隔は掛ける前に割る。31日の月は2.68e15ナノ秒で、Durationの実体で
	// あるint64が届くのは9.22e18までであるため、先に掛ける形は件数が3,400ほど
	// を超えたところであふれる。ここへ渡るのは1か月分の取り分 (実行1回分の生成量
	// で130件ほど) であり、今はどちらの順序でもあふれないが、割ってから掛ける形は、
	// amountsの件数をどう増やしてもそれが成り立ち続けるようにする。
	interval := end.Sub(start) / time.Duration(count+1)

	return start.Add(interval * time.Duration(index+1))
}
