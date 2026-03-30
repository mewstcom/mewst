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
	queries *query.Queries
}

// NewProfileRepository はProfileRepositoryを生成する
func NewProfileRepository(db query.DBTX) *ProfileRepository {
	return &ProfileRepository{
		queries: query.New(db),
	}
}

// WithTx はトランザクションを設定したProfileRepositoryを返す
func (r *ProfileRepository) WithTx(tx *sql.Tx) *ProfileRepository {
	return &ProfileRepository{
		queries: r.queries.WithTx(tx),
	}
}

// CreateProfileParams はプロフィール作成のパラメータ
type CreateProfileParams struct {
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

// GetByID はIDでプロフィールを取得する
func (r *ProfileRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Profile, error) {
	row, err := r.queries.GetProfileByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return r.toModel(row.ID, row.OwnerType, row.Atname, row.Name, row.Description, row.ImageUrl,
		row.JoinedAt, row.AvatarKind, row.GravatarEmail, row.GravatarUrl,
		row.DiscardedAt, row.LastPostAt, row.CreatedAt, row.UpdatedAt), nil
}

// GetByAtname はアットネームでプロフィールを取得する
func (r *ProfileRepository) GetByAtname(ctx context.Context, atname string) (*model.Profile, error) {
	row, err := r.queries.GetProfileByAtname(ctx, atname)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return r.toModel(row.ID, row.OwnerType, row.Atname, row.Name, row.Description, row.ImageUrl,
		row.JoinedAt, row.AvatarKind, row.GravatarEmail, row.GravatarUrl,
		row.DiscardedAt, row.LastPostAt, row.CreatedAt, row.UpdatedAt), nil
}

// ExistsByAtname はアットネームでプロフィールの存在を確認する
func (r *ProfileRepository) ExistsByAtname(ctx context.Context, atname string) (bool, error) {
	return r.queries.ExistsProfileByAtname(ctx, atname)
}

// Create はプロフィールを作成する
func (r *ProfileRepository) Create(ctx context.Context, params CreateProfileParams) (*model.Profile, error) {
	row, err := r.queries.CreateProfile(ctx, query.CreateProfileParams{
		OwnerType:     params.OwnerType,
		Atname:        params.Atname,
		Name:          params.Name,
		Description:   params.Description,
		ImageUrl:      params.ImageURL,
		JoinedAt:      params.JoinedAt,
		AvatarKind:    params.AvatarKind,
		GravatarEmail: params.GravatarEmail,
		GravatarUrl:   params.GravatarURL,
	})
	if err != nil {
		return nil, err
	}

	return r.toModel(row.ID, row.OwnerType, row.Atname, row.Name, row.Description, row.ImageUrl,
		row.JoinedAt, row.AvatarKind, row.GravatarEmail, row.GravatarUrl,
		row.DiscardedAt, row.LastPostAt, row.CreatedAt, row.UpdatedAt), nil
}

// toModel はsqlcの行データをモデルに変換する
func (r *ProfileRepository) toModel(
	id uuid.UUID, ownerType, atname, name, description, imageURL string,
	joinedAt time.Time, avatarKind, gravatarEmail, gravatarURL string,
	discardedAt, lastPostAt sql.NullTime, createdAt, updatedAt time.Time,
) *model.Profile {
	var discardedAtPtr *time.Time
	if discardedAt.Valid {
		discardedAtPtr = &discardedAt.Time
	}

	var lastPostAtPtr *time.Time
	if lastPostAt.Valid {
		lastPostAtPtr = &lastPostAt.Time
	}

	return &model.Profile{
		ID:            id,
		OwnerType:     ownerType,
		Atname:        atname,
		Name:          name,
		Description:   description,
		ImageURL:      imageURL,
		JoinedAt:      joinedAt,
		AvatarKind:    avatarKind,
		GravatarEmail: gravatarEmail,
		GravatarURL:   gravatarURL,
		DiscardedAt:   discardedAtPtr,
		LastPostAt:    lastPostAtPtr,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
