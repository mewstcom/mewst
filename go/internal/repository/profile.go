package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// ProfileRepository はプロフィールのリポジトリ
type ProfileRepository struct {
	q *query.Queries
}

// NewProfileRepository はProfileRepositoryを生成する
func NewProfileRepository(q *query.Queries) *ProfileRepository {
	return &ProfileRepository{q: q}
}

// WithTx はトランザクションを設定したProfileRepositoryを返す
func (r *ProfileRepository) WithTx(tx *sql.Tx) *ProfileRepository {
	return &ProfileRepository{q: r.q.WithTx(tx)}
}

// CreateProfileInput はプロフィール作成の入力パラメータ
type CreateProfileInput struct {
	OwnerType     string
	Atname        string
	Name          string
	Description   string
	ImageURL      string
	JoinedAt      time.Time
	AvatarKind    string
	GravatarEmail string
	GravatarURL   string
}

// FindByID はIDでプロフィールを取得する
func (r *ProfileRepository) FindByID(ctx context.Context, id model.ProfileID) (*model.Profile, error) {
	row, err := r.q.GetProfileByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toProfileModel(row), nil
}

// FindByAtname はアットネームでプロフィールを取得する
func (r *ProfileRepository) FindByAtname(ctx context.Context, atname string) (*model.Profile, error) {
	row, err := r.q.GetProfileByAtname(ctx, atname)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toProfileModel(row), nil
}

// ExistsByAtname はアットネームでプロフィールの存在を確認する
func (r *ProfileRepository) ExistsByAtname(ctx context.Context, atname string) (bool, error) {
	return r.q.ExistsProfileByAtname(ctx, atname)
}

// UpdateLastPostAt updates the profile's last_post_at.
// [Ja] UpdateLastPostAt はプロフィールの last_post_at を更新する。
func (r *ProfileRepository) UpdateLastPostAt(ctx context.Context, id model.ProfileID, lastPostAt time.Time) error {
	return r.q.UpdateProfileLastPostAt(ctx, query.UpdateProfileLastPostAtParams{
		ID:         uuid.UUID(id),
		LastPostAt: sql.NullTime{Time: lastPostAt, Valid: true},
	})
}

// Create はプロフィールを作成する
func (r *ProfileRepository) Create(ctx context.Context, input CreateProfileInput) (*model.Profile, error) {
	row, err := r.q.CreateProfile(ctx, query.CreateProfileParams{
		OwnerType:     input.OwnerType,
		Atname:        input.Atname,
		Name:          input.Name,
		Description:   input.Description,
		ImageUrl:      input.ImageURL,
		JoinedAt:      input.JoinedAt,
		AvatarKind:    input.AvatarKind,
		GravatarEmail: input.GravatarEmail,
		GravatarUrl:   input.GravatarURL,
	})
	if err != nil {
		return nil, err
	}
	return toProfileModel(row), nil
}

// toProfileModel converts a query.Profile row into a model.Profile. It is a
// package-private free function so SessionRepository's JOIN-based auth lookup
// can reuse the conversion without instantiating a ProfileRepository.
//
// [Ja] toProfileModel は query.Profile を model.Profile に変換するパッケージ非公開の
// 自由関数。SessionRepository が JOIN で取得した profile 行を ProfileRepository
// なしで変換できるように、メソッドではなく自由関数にしている。
func toProfileModel(row query.Profile) *model.Profile {
	var discardedAt *time.Time
	if row.DiscardedAt.Valid {
		discardedAt = &row.DiscardedAt.Time
	}

	var lastPostAt *time.Time
	if row.LastPostAt.Valid {
		lastPostAt = &row.LastPostAt.Time
	}

	return &model.Profile{
		ID:            model.ProfileID(row.ID),
		OwnerType:     row.OwnerType,
		Atname:        row.Atname,
		Name:          row.Name,
		Description:   row.Description,
		ImageURL:      row.ImageUrl,
		JoinedAt:      row.JoinedAt,
		AvatarKind:    row.AvatarKind,
		GravatarEmail: row.GravatarEmail,
		GravatarURL:   row.GravatarUrl,
		DiscardedAt:   discardedAt,
		LastPostAt:    lastPostAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
