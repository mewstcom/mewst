package seed

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// exportState is one export a run leaves behind: which role requested it, and
// the status it is left in.
//
// [Ja] exportState は、実行 1 回分が残すエクスポート 1 件。どの役割が申請したのかと、
// どの状態で残されるのか。
type exportState struct {
	role   seedRole
	status model.ExportStatus
}

// exportStates is which role holds an export in which status.
//
// The statuses are spread over three roles rather than given to one. A profile
// is allowed at most one queued or started export by a partial unique index,
// so the states an export screen is read in cannot all be reached from a
// single account.
//
// succeeded is not among them. What that status offers is a download of an
// archive in the object storage, and a row written without one is a screen
// whose download fails. It is reached by running an export from the screen,
// which is what the failed row of roleMain leads to.
//
// [Ja] exportStates は、どの役割がどの状態のエクスポートを持つか。
//
// 状態は 1 つの役割へ与えず 3 つの役割へ散らす。プロフィールが持てる queued /
// started のエクスポートは部分ユニークインデックスにより 1 件までであり、エクスポートの
// 画面を読むための状態を、1 つのアカウントからすべて辿ることはできないため。
//
// succeeded はここに無い。その状態が提示するのはオブジェクトストレージにある
// アーカイブのダウンロードであり、それを持たずに書かれた行は、ダウンロードが失敗する
// 画面になる。この状態へは画面からエクスポートを実行して辿り着くのであり、roleMain の
// failed の行が導くのがそれである。
var exportStates = []exportState{
	{role: roleMain, status: model.ExportStatusFailed},
	{role: roleFollower, status: model.ExportStatusQueued},
	{role: roleEnglish, status: model.ExportStatusStarted},
}

// exportWriter writes the exports of one run.
//
// [Ja] exportWriter は、実行 1 回分のエクスポートを書き込む。
type exportWriter struct {
	tx      *sql.Tx
	exports *repository.ExportRepository
}

// createExports writes one export per role in exportStates and moves it into
// the status that role is there to show.
//
// It runs after everything that writes a post. Creating an export fixes the
// profile's kept posts into export_posts in the same statement, so an export
// written before the posts would stand for an archive of nothing.
//
// [Ja] createExports は、exportStates の役割ごとに 1 件のエクスポートを書き込み、
// その役割が示すためにいる状態へ遷移させる。
//
// ポストを書き込むもののすべてより後に実行する。エクスポートの作成は、プロフィールの
// 残っているポストを同じ文で export_posts へ固定するため、ポストより前に書かれた
// エクスポートは、中身の無いアーカイブを表す行になる。
func createExports(ctx context.Context, tx *sql.Tx, accounts []seedAccount) error {
	writer := &exportWriter{
		tx:      tx,
		exports: repository.NewExportRepository(query.New(tx)),
	}

	// The roles are walked in the roster's own order rather than in the order
	// exportStates happens to name them, so that which account is written
	// first is decided in one place for every generator.
	//
	// [Ja] 役割は、exportStates がたまたま名指しする順ではなく名簿自身の順で辿る。
	// どのアカウントを先に書くのかが、どの生成器についても 1 箇所で決まるようにする
	// ため。
	for _, role := range allSeedRoles {
		state, ok := exportStateForRole(role)
		if !ok {
			continue
		}

		account, err := accountForRole(accounts, role)
		if err != nil {
			return err
		}

		if err := writer.writeExport(ctx, account, state.status); err != nil {
			return fmt.Errorf("役割 %s の %s のエクスポートの作成に失敗: %w", role, state.status, err)
		}
	}

	return nil
}

// exportStateForRole returns the export role holds, and whether it holds one
// at all.
//
// [Ja] exportStateForRole は、role が持つエクスポートと、そもそもそれを持つのかを
// 返す。
func exportStateForRole(role seedRole) (exportState, bool) {
	for _, state := range exportStates {
		if state.role == role {
			return state, true
		}
	}

	return exportState{}, false
}

// writeExport creates one account's export and leaves it in status.
//
// [Ja] writeExport は、あるアカウントのエクスポートを 1 件作成し、status の状態で
// 残す。
func (w *exportWriter) writeExport(ctx context.Context, account seedAccount, status model.ExportStatus) error {
	export, err := w.exports.Create(ctx, repository.CreateExportInput{
		ProfileID: account.profile.ID,
		ActorID:   account.actor.ID,
	})
	if err != nil {
		return fmt.Errorf("エクスポートの作成に失敗: %w", err)
	}

	// No row back means the profile is past the boundary its deletion
	// establishes, which no account of a run has crossed. It is reported
	// rather than skipped, because an export the run meant to leave behind is
	// missing from a screen either way, and only one of the two says why.
	//
	// [Ja] 行が返らないのは、プロフィールがその削除の確立する境界を越えている
	// 場合であり、実行のどのアカウントもそれを越えていない。読み飛ばさずに報告するのは、
	// 実行が残すつもりだったエクスポートがどちらにせよ画面から欠けるのであり、その理由を
	// 述べるのは 2 つのうち一方だけであるため。
	if export == nil {
		return fmt.Errorf("プロフィールの削除が始まっているためエクスポートを作成できません")
	}

	if err := w.requireSnapshot(ctx, export.ID); err != nil {
		return err
	}

	return w.transition(ctx, export, status)
}

// requireSnapshot reports an export whose snapshot came out empty.
//
// The snapshot is what the archive is built from, and it is fixed by the same
// statement that creates the export. An export created before its profile's
// posts exist holds a snapshot of nothing, which reconciliation picks up and
// turns into an empty zip rather than into the archive the row stands for.
//
// It is read back here rather than after the transitions, because reaching a
// terminal status discards the snapshot: from there on, an export that never
// had one and an export that has finished with it are the same row.
//
// [Ja] requireSnapshot は、snapshot が空になったエクスポートを報告する。
//
// snapshot はアーカイブの生成元であり、エクスポートを作成するのと同じ文で固定される。
// プロフィールのポストより前に作成されたエクスポートは中身の無い snapshot を持ち、
// それはリコンシリエーションに拾われて、その行が表すアーカイブではなく空の zip になる。
//
// 遷移の後ではなくここで読み戻すのは、終端状態への到達が snapshot を破棄するため。
// そこから先では、snapshot を一度も持たなかったエクスポートと、それを使い終えた
// エクスポートは同じ行になる。
func (w *exportWriter) requireSnapshot(ctx context.Context, id model.ExportID) error {
	var count int

	if err := w.tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM export_posts WHERE export_id = $1
	`, uuid.UUID(id)).Scan(&count); err != nil {
		return fmt.Errorf("エクスポートの snapshot の件数の取得に失敗: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("エクスポートの snapshot に含まれるポストが 1 件もありません")
	}

	return nil
}

// transition moves a created export into status along the path the generation
// job takes to it.
//
// Every status but queued is reached through started, which is what the job
// does: it stamps the attempt before it builds anything, and a row placed into
// a later status by a statement of the seed's own would carry neither the
// attempt count nor the started_at a screen displays.
//
// [Ja] transition は、作成済みのエクスポートを、生成ジョブがその状態へ辿る経路に
// 沿って status へ遷移させる。
//
// queued 以外のどの状態へも started を経て到達する。それが生成ジョブの行うこと
// だからである。ジョブは何かを組み立てるより前に試行を刻むのであり、シード独自の文で
// 後の状態へ置かれた行は、画面が表示する試行回数も started_at も持たないことになる。
func (w *exportWriter) transition(ctx context.Context, export *model.Export, status model.ExportStatus) error {
	switch status {
	case model.ExportStatusQueued:
		return nil
	case model.ExportStatusStarted:
		_, err := w.markStarted(ctx, export)
		return err
	case model.ExportStatusFailed:
		started, err := w.markStarted(ctx, export)
		if err != nil {
			return err
		}

		return w.markFailed(ctx, started)
	default:
		return fmt.Errorf("エクスポートの状態 %s はシードが作れません", status)
	}
}

// markStarted moves the export into started and returns the row the
// transition produced, whose updated_at is the token the next transition has
// to present.
//
// [Ja] markStarted は、エクスポートを started へ遷移させ、その遷移が生んだ行を返す。
// その updated_at が、次の遷移が提示するトークンになる。
func (w *exportWriter) markStarted(ctx context.Context, export *model.Export) (*model.Export, error) {
	started, err := w.exports.MarkStarted(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("エクスポートの started への遷移に失敗: %w", err)
	}

	// A guard that did not match is a conflict with another attempt, which a
	// run has none of: it holds the only transaction that has seen this row.
	// It is reported rather than passed over, because the export would
	// otherwise stay in the status before it while the run reports success.
	//
	// [Ja] ガードが一致しないのは別の試行との競合だが、実行にそれは無い。この行を
	// 見たことのある唯一のトランザクションを実行自身が持っているためである。見過ごさずに
	// 報告するのは、そうしなければエクスポートが手前の状態に留まったまま、実行が成功を
	// 報告することになるため。
	if started == nil {
		return nil, fmt.Errorf("エクスポートの started への遷移が競合しました")
	}

	return started, nil
}

// markFailed moves a started export into failed, which is where the export of
// roleMain is left: the screen it produces is the one a re-run is started
// from.
//
// [Ja] markFailed は、started のエクスポートを failed へ遷移させる。roleMain の
// エクスポートが残されるのがこの状態であり、それが作る画面が、再実行を始める画面に
// なる。
func (w *exportWriter) markFailed(ctx context.Context, export *model.Export) error {
	marked, err := w.exports.MarkFailed(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		return fmt.Errorf("エクスポートの failed への遷移に失敗: %w", err)
	}

	if !marked {
		return fmt.Errorf("エクスポートの failed への遷移が競合しました")
	}

	return nil
}
