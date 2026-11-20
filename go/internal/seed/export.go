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

// exportStateは、実行1回分が残すエクスポート1件。どの役割が申請したのかと、
// どの状態で残されるのか。
type exportState struct {
	role   seedRole
	status model.ExportStatus
}

// exportStatesは、どの役割がどの状態のエクスポートを持つか。
//
// 状態は1つの役割へ与えず3つの役割へ散らす。プロフィールが持てるqueued /
// startedのエクスポートは部分ユニークインデックスにより1件までであり、エクスポートの
// 画面を読むための状態を、1つのアカウントからすべて辿ることはできないため。
//
// succeededはここに無い。その状態が提示するのはオブジェクトストレージにある
// アーカイブのダウンロードであり、それを持たずに書かれた行は、ダウンロードが失敗する
// 画面になる。この状態へは画面からエクスポートを実行して辿り着くのであり、roleMainの
// failedの行が導くのがそれである。
var exportStates = []exportState{
	{role: roleMain, status: model.ExportStatusFailed},
	{role: roleFollower, status: model.ExportStatusQueued},
	{role: roleEnglish, status: model.ExportStatusStarted},
}

// exportWriterは、実行1回分のエクスポートを書き込む。
type exportWriter struct {
	tx      *sql.Tx
	exports *repository.ExportRepository
}

// createExportsは、exportStatesの役割ごとに1件のエクスポートを書き込み、
// その役割が示すためにいる状態へ遷移させる。
//
// ポストを書き込むもののすべてより後に実行する。エクスポートの作成は、プロフィールの
// 残っているポストを同じ文でexport_postsへ固定するため、ポストより前に書かれた
// エクスポートは、中身の無いアーカイブを表す行になる。
func createExports(ctx context.Context, tx *sql.Tx, accounts []seedAccount) error {
	writer := &exportWriter{
		tx:      tx,
		exports: repository.NewExportRepository(query.New(tx)),
	}

	// 役割は、exportStatesがたまたま名指しする順ではなく名簿自身の順で辿る。
	// どのアカウントを先に書くのかが、どの生成器についても1箇所で決まるようにする
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
			return fmt.Errorf("役割%sの%sのエクスポートの作成に失敗: %w", role, state.status, err)
		}
	}

	return nil
}

// exportStateForRoleは、roleが持つエクスポートと、そもそもそれを持つのかを
// 返す。
func exportStateForRole(role seedRole) (exportState, bool) {
	for _, state := range exportStates {
		if state.role == role {
			return state, true
		}
	}

	return exportState{}, false
}

// writeExportは、あるアカウントのエクスポートを1件作成し、statusの状態で
// 残す。
func (w *exportWriter) writeExport(ctx context.Context, account seedAccount, status model.ExportStatus) error {
	export, err := w.exports.Create(ctx, repository.CreateExportInput{
		ProfileID: account.profile.ID,
		ActorID:   account.actor.ID,
	})
	if err != nil {
		return fmt.Errorf("エクスポートの作成に失敗: %w", err)
	}

	// 行が返らないのは、プロフィールがその削除の確立する境界を越えている
	// 場合であり、実行のどのアカウントもそれを越えていない。読み飛ばさずに報告するのは、
	// 実行が残すつもりだったエクスポートがどちらにせよ画面から欠けるのであり、その理由を
	// 述べるのは2つのうち一方だけであるため。
	if export == nil {
		return fmt.Errorf("プロフィールの削除が始まっているためエクスポートを作成できません")
	}

	if err := w.requireSnapshot(ctx, export.ID); err != nil {
		return err
	}

	return w.transition(ctx, export, status)
}

// requireSnapshotは、snapshotが空になったエクスポートを報告する。
//
// snapshotはアーカイブの生成元であり、エクスポートを作成するのと同じ文で固定される。
// プロフィールのポストより前に作成されたエクスポートは中身の無いsnapshotを持ち、
// それはリコンシリエーションに拾われて、その行が表すアーカイブではなく空のzipになる。
//
// 遷移の後ではなくここで読み戻すのは、終端状態への到達がsnapshotを破棄するため。
// そこから先では、snapshotを一度も持たなかったエクスポートと、それを使い終えた
// エクスポートは同じ行になる。
func (w *exportWriter) requireSnapshot(ctx context.Context, id model.ExportID) error {
	var count int

	if err := w.tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM export_posts WHERE export_id = $1
	`, uuid.UUID(id)).Scan(&count); err != nil {
		return fmt.Errorf("エクスポートのsnapshotの件数の取得に失敗: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("エクスポートのsnapshotに含まれるポストが1件もありません")
	}

	return nil
}

// transitionは、作成済みのエクスポートを、生成ジョブがその状態へ辿る経路に
// 沿ってstatusへ遷移させる。
//
// queued以外のどの状態へもstartedを経て到達する。それが生成ジョブの行うこと
// だからである。ジョブは何かを組み立てるより前に試行を刻むのであり、シード独自の文で
// 後の状態へ置かれた行は、画面が表示する試行回数もstarted_atも持たないことになる。
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
		return fmt.Errorf("エクスポートの状態%sはシードが作れません", status)
	}
}

// markStartedは、エクスポートをstartedへ遷移させ、その遷移が生んだ行を返す。
// そのupdated_atが、次の遷移が提示するトークンになる。
func (w *exportWriter) markStarted(ctx context.Context, export *model.Export) (*model.Export, error) {
	started, err := w.exports.MarkStarted(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("エクスポートのstartedへの遷移に失敗: %w", err)
	}

	// ガードが一致しないのは別の試行との競合だが、実行にそれは無い。この行を
	// 見たことのある唯一のトランザクションを実行自身が持っているためである。見過ごさずに
	// 報告するのは、そうしなければエクスポートが手前の状態に留まったまま、実行が成功を
	// 報告することになるため。
	if started == nil {
		return nil, fmt.Errorf("エクスポートのstartedへの遷移が競合しました")
	}

	return started, nil
}

// markFailedは、startedのエクスポートをfailedへ遷移させる。roleMainの
// エクスポートが残されるのがこの状態であり、それが作る画面が、再実行を始める画面に
// なる。
func (w *exportWriter) markFailed(ctx context.Context, export *model.Export) error {
	marked, err := w.exports.MarkFailed(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		return fmt.Errorf("エクスポートのfailedへの遷移に失敗: %w", err)
	}

	if !marked {
		return fmt.Errorf("エクスポートのfailedへの遷移が競合しました")
	}

	return nil
}
