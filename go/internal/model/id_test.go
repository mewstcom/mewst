package model_test

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
)

func TestUserID_String(t *testing.T) {
	t.Parallel()

	u := uuid.New()
	id := model.UserID(u)

	if got := id.String(); got != u.String() {
		t.Errorf("UserID.String() = %q, want %q", got, u.String())
	}
}

func TestUserIDsToUUIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ids  []model.UserID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			ids:  nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			ids:  []model.UserID{},
		},
		{
			name: "単一要素を変換できる",
			ids:  []model.UserID{model.UserID(uuid.New())},
		},
		{
			name: "複数要素を順序を保って変換できる",
			ids: []model.UserID{
				model.UserID(uuid.New()),
				model.UserID(uuid.New()),
				model.UserID(uuid.New()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.UserIDsToUUIDs(tt.ids)

			if len(got) != len(tt.ids) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.ids))
			}
			for i, id := range tt.ids {
				if got[i] != uuid.UUID(id) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], uuid.UUID(id))
				}
			}
		})
	}
}

func TestUUIDsToUserIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		us   []uuid.UUID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			us:   nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			us:   []uuid.UUID{},
		},
		{
			name: "単一要素を変換できる",
			us:   []uuid.UUID{uuid.New()},
		},
		{
			name: "複数要素を順序を保って変換できる",
			us:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.UUIDsToUserIDs(tt.us)

			if len(got) != len(tt.us) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.us))
			}
			for i, u := range tt.us {
				if got[i] != model.UserID(u) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], model.UserID(u))
				}
			}
		})
	}
}

func TestUserIDsAndUUIDsRoundTrip(t *testing.T) {
	t.Parallel()

	src := []model.UserID{
		model.UserID(uuid.New()),
		model.UserID(uuid.New()),
		model.UserID(uuid.New()),
	}

	got := model.UUIDsToUserIDs(model.UserIDsToUUIDs(src))

	if !reflect.DeepEqual(src, got) {
		t.Errorf("round trip mismatch: got %v want %v", got, src)
	}
}

func TestProfileID_String(t *testing.T) {
	t.Parallel()

	u := uuid.New()
	id := model.ProfileID(u)

	if got := id.String(); got != u.String() {
		t.Errorf("ProfileID.String() = %q, want %q", got, u.String())
	}
}

func TestProfileIDsToUUIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ids  []model.ProfileID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			ids:  nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			ids:  []model.ProfileID{},
		},
		{
			name: "単一要素を変換できる",
			ids:  []model.ProfileID{model.ProfileID(uuid.New())},
		},
		{
			name: "複数要素を順序を保って変換できる",
			ids: []model.ProfileID{
				model.ProfileID(uuid.New()),
				model.ProfileID(uuid.New()),
				model.ProfileID(uuid.New()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.ProfileIDsToUUIDs(tt.ids)

			if len(got) != len(tt.ids) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.ids))
			}
			for i, id := range tt.ids {
				if got[i] != uuid.UUID(id) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], uuid.UUID(id))
				}
			}
		})
	}
}

func TestUUIDsToProfileIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		us   []uuid.UUID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			us:   nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			us:   []uuid.UUID{},
		},
		{
			name: "単一要素を変換できる",
			us:   []uuid.UUID{uuid.New()},
		},
		{
			name: "複数要素を順序を保って変換できる",
			us:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.UUIDsToProfileIDs(tt.us)

			if len(got) != len(tt.us) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.us))
			}
			for i, u := range tt.us {
				if got[i] != model.ProfileID(u) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], model.ProfileID(u))
				}
			}
		})
	}
}

func TestProfileIDsAndUUIDsRoundTrip(t *testing.T) {
	t.Parallel()

	src := []model.ProfileID{
		model.ProfileID(uuid.New()),
		model.ProfileID(uuid.New()),
		model.ProfileID(uuid.New()),
	}

	got := model.UUIDsToProfileIDs(model.ProfileIDsToUUIDs(src))

	if !reflect.DeepEqual(src, got) {
		t.Errorf("round trip mismatch: got %v want %v", got, src)
	}
}

func TestActorID_String(t *testing.T) {
	t.Parallel()

	u := uuid.New()
	id := model.ActorID(u)

	if got := id.String(); got != u.String() {
		t.Errorf("ActorID.String() = %q, want %q", got, u.String())
	}
}

func TestActorIDsToUUIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ids  []model.ActorID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			ids:  nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			ids:  []model.ActorID{},
		},
		{
			name: "単一要素を変換できる",
			ids:  []model.ActorID{model.ActorID(uuid.New())},
		},
		{
			name: "複数要素を順序を保って変換できる",
			ids: []model.ActorID{
				model.ActorID(uuid.New()),
				model.ActorID(uuid.New()),
				model.ActorID(uuid.New()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.ActorIDsToUUIDs(tt.ids)

			if len(got) != len(tt.ids) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.ids))
			}
			for i, id := range tt.ids {
				if got[i] != uuid.UUID(id) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], uuid.UUID(id))
				}
			}
		})
	}
}

func TestUUIDsToActorIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		us   []uuid.UUID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			us:   nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			us:   []uuid.UUID{},
		},
		{
			name: "単一要素を変換できる",
			us:   []uuid.UUID{uuid.New()},
		},
		{
			name: "複数要素を順序を保って変換できる",
			us:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.UUIDsToActorIDs(tt.us)

			if len(got) != len(tt.us) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.us))
			}
			for i, u := range tt.us {
				if got[i] != model.ActorID(u) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], model.ActorID(u))
				}
			}
		})
	}
}

func TestActorIDsAndUUIDsRoundTrip(t *testing.T) {
	t.Parallel()

	src := []model.ActorID{
		model.ActorID(uuid.New()),
		model.ActorID(uuid.New()),
		model.ActorID(uuid.New()),
	}

	got := model.UUIDsToActorIDs(model.ActorIDsToUUIDs(src))

	if !reflect.DeepEqual(src, got) {
		t.Errorf("round trip mismatch: got %v want %v", got, src)
	}
}

func TestSessionID_String(t *testing.T) {
	t.Parallel()

	u := uuid.New()
	id := model.SessionID(u)

	if got := id.String(); got != u.String() {
		t.Errorf("SessionID.String() = %q, want %q", got, u.String())
	}
}

func TestSessionIDsToUUIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ids  []model.SessionID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			ids:  nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			ids:  []model.SessionID{},
		},
		{
			name: "単一要素を変換できる",
			ids:  []model.SessionID{model.SessionID(uuid.New())},
		},
		{
			name: "複数要素を順序を保って変換できる",
			ids: []model.SessionID{
				model.SessionID(uuid.New()),
				model.SessionID(uuid.New()),
				model.SessionID(uuid.New()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.SessionIDsToUUIDs(tt.ids)

			if len(got) != len(tt.ids) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.ids))
			}
			for i, id := range tt.ids {
				if got[i] != uuid.UUID(id) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], uuid.UUID(id))
				}
			}
		})
	}
}

func TestUUIDsToSessionIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		us   []uuid.UUID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			us:   nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			us:   []uuid.UUID{},
		},
		{
			name: "単一要素を変換できる",
			us:   []uuid.UUID{uuid.New()},
		},
		{
			name: "複数要素を順序を保って変換できる",
			us:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.UUIDsToSessionIDs(tt.us)

			if len(got) != len(tt.us) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.us))
			}
			for i, u := range tt.us {
				if got[i] != model.SessionID(u) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], model.SessionID(u))
				}
			}
		})
	}
}

func TestSessionIDsAndUUIDsRoundTrip(t *testing.T) {
	t.Parallel()

	src := []model.SessionID{
		model.SessionID(uuid.New()),
		model.SessionID(uuid.New()),
		model.SessionID(uuid.New()),
	}

	got := model.UUIDsToSessionIDs(model.SessionIDsToUUIDs(src))

	if !reflect.DeepEqual(src, got) {
		t.Errorf("round trip mismatch: got %v want %v", got, src)
	}
}

func TestEmailConfirmationID_String(t *testing.T) {
	t.Parallel()

	u := uuid.New()
	id := model.EmailConfirmationID(u)

	if got := id.String(); got != u.String() {
		t.Errorf("EmailConfirmationID.String() = %q, want %q", got, u.String())
	}
}

func TestEmailConfirmationIDsToUUIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ids  []model.EmailConfirmationID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			ids:  nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			ids:  []model.EmailConfirmationID{},
		},
		{
			name: "単一要素を変換できる",
			ids:  []model.EmailConfirmationID{model.EmailConfirmationID(uuid.New())},
		},
		{
			name: "複数要素を順序を保って変換できる",
			ids: []model.EmailConfirmationID{
				model.EmailConfirmationID(uuid.New()),
				model.EmailConfirmationID(uuid.New()),
				model.EmailConfirmationID(uuid.New()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.EmailConfirmationIDsToUUIDs(tt.ids)

			if len(got) != len(tt.ids) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.ids))
			}
			for i, id := range tt.ids {
				if got[i] != uuid.UUID(id) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], uuid.UUID(id))
				}
			}
		})
	}
}

func TestUUIDsToEmailConfirmationIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		us   []uuid.UUID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			us:   nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			us:   []uuid.UUID{},
		},
		{
			name: "単一要素を変換できる",
			us:   []uuid.UUID{uuid.New()},
		},
		{
			name: "複数要素を順序を保って変換できる",
			us:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.UUIDsToEmailConfirmationIDs(tt.us)

			if len(got) != len(tt.us) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.us))
			}
			for i, u := range tt.us {
				if got[i] != model.EmailConfirmationID(u) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], model.EmailConfirmationID(u))
				}
			}
		})
	}
}

func TestEmailConfirmationIDsAndUUIDsRoundTrip(t *testing.T) {
	t.Parallel()

	src := []model.EmailConfirmationID{
		model.EmailConfirmationID(uuid.New()),
		model.EmailConfirmationID(uuid.New()),
		model.EmailConfirmationID(uuid.New()),
	}

	got := model.UUIDsToEmailConfirmationIDs(model.EmailConfirmationIDsToUUIDs(src))

	if !reflect.DeepEqual(src, got) {
		t.Errorf("round trip mismatch: got %v want %v", got, src)
	}
}

func TestUserProfileID_String(t *testing.T) {
	t.Parallel()

	u := uuid.New()
	id := model.UserProfileID(u)

	if got := id.String(); got != u.String() {
		t.Errorf("UserProfileID.String() = %q, want %q", got, u.String())
	}
}

func TestUserProfileIDsToUUIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ids  []model.UserProfileID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			ids:  nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			ids:  []model.UserProfileID{},
		},
		{
			name: "単一要素を変換できる",
			ids:  []model.UserProfileID{model.UserProfileID(uuid.New())},
		},
		{
			name: "複数要素を順序を保って変換できる",
			ids: []model.UserProfileID{
				model.UserProfileID(uuid.New()),
				model.UserProfileID(uuid.New()),
				model.UserProfileID(uuid.New()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.UserProfileIDsToUUIDs(tt.ids)

			if len(got) != len(tt.ids) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.ids))
			}
			for i, id := range tt.ids {
				if got[i] != uuid.UUID(id) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], uuid.UUID(id))
				}
			}
		})
	}
}

func TestUUIDsToUserProfileIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		us   []uuid.UUID
	}{
		{
			name: "nil スライスは長さ 0 のスライスを返す",
			us:   nil,
		},
		{
			name: "空スライスは長さ 0 のスライスを返す",
			us:   []uuid.UUID{},
		},
		{
			name: "単一要素を変換できる",
			us:   []uuid.UUID{uuid.New()},
		},
		{
			name: "複数要素を順序を保って変換できる",
			us:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.UUIDsToUserProfileIDs(tt.us)

			if len(got) != len(tt.us) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.us))
			}
			for i, u := range tt.us {
				if got[i] != model.UserProfileID(u) {
					t.Errorf("got[%d] = %v, want %v", i, got[i], model.UserProfileID(u))
				}
			}
		})
	}
}

func TestUserProfileIDsAndUUIDsRoundTrip(t *testing.T) {
	t.Parallel()

	src := []model.UserProfileID{
		model.UserProfileID(uuid.New()),
		model.UserProfileID(uuid.New()),
		model.UserProfileID(uuid.New()),
	}

	got := model.UUIDsToUserProfileIDs(model.UserProfileIDsToUUIDs(src))

	if !reflect.DeepEqual(src, got) {
		t.Errorf("round trip mismatch: got %v want %v", got, src)
	}
}
