package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// GetSettingIndexUsecase reads which feature-flagged entries the settings menu
// (GET /settings) offers the signed-in actor. The rest of the menu is static,
// so this exists to keep the export entry out of the menu until its flag lets
// the actor reach the export page at all.
//
// [Ja] GetSettingIndexUsecase は設定メニュー (GET /settings) がログイン中の actor に
// 提供する、フィーチャーフラグで制御する項目を取得する。メニューの他の項目は静的で
// あり、本 UseCase はフラグによって actor がエクスポート画面に到達できるようになる
// まで、エクスポート項目をメニューから外しておくために存在する。
type GetSettingIndexUsecase struct {
	featureFlagRepo *repository.FeatureFlagRepository
}

// NewGetSettingIndexUsecase creates a GetSettingIndexUsecase.
//
// [Ja] NewGetSettingIndexUsecase は GetSettingIndexUsecase を生成する。
func NewGetSettingIndexUsecase(featureFlagRepo *repository.FeatureFlagRepository) *GetSettingIndexUsecase {
	return &GetSettingIndexUsecase{
		featureFlagRepo: featureFlagRepo,
	}
}

// GetSettingIndexInput holds the input parameters for reading the settings
// menu. ActorID comes from the session and identifies who the menu is being
// rendered for.
//
// [Ja] GetSettingIndexInput は設定メニューの取得の入力パラメータ。ActorID は
// セッション由来で、誰に向けてメニューを描画するのかを表す。
type GetSettingIndexInput struct {
	ActorID model.ActorID
}

// GetSettingIndexOutput is the result of reading the settings menu.
// ExportEnabled reports whether the export entry belongs in it.
//
// [Ja] GetSettingIndexOutput は設定メニューの取得結果。ExportEnabled はエクスポート
// 項目をメニューに含めるかどうかを表す。
type GetSettingIndexOutput struct {
	ExportEnabled bool
}

// Execute reports whether the actor has the export feature flag.
//
// The decision is made per actor, not per device. The reverse proxy also
// accepts a device_token grant when it routes /settings/export, but that grant
// identifies a browser regardless of who is signed in, while this menu is
// rendered for one known actor. Rollout grants the flag per actor, so an actor
// who can use the export page is offered the entry, and an actor who cannot is
// not shown a link that would leave them on the Rails 404.
//
// [Ja] Execute は actor がエクスポートのフィーチャーフラグを持つかを返す。
//
// 判定はデバイス単位ではなく actor 単位で行う。リバースプロキシは
// /settings/export の振り分けで device_token による付与も受け付けるが、その付与は
// 誰がログインしているかに依らずブラウザを識別するものであり、このメニューは特定の
// actor に向けて描画される。公開はフラグを actor 単位で付与して進めるため、
// エクスポート画面を使える actor には項目を提供し、使えない actor には Rails の 404
// に行き着くリンクを見せない。
func (uc *GetSettingIndexUsecase) Execute(ctx context.Context, input GetSettingIndexInput) (*GetSettingIndexOutput, error) {
	enabled, err := uc.featureFlagRepo.IsEnabledForActor(ctx, input.ActorID, model.FeatureFlagExport)
	if err != nil {
		return nil, fmt.Errorf("エクスポートのフィーチャーフラグの判定に失敗: %w", err)
	}

	return &GetSettingIndexOutput{ExportEnabled: enabled}, nil
}
