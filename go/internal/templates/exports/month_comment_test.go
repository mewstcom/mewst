package exports

import "testing"

func TestCommentText_TakesOutWhatWouldEndOrEscapeTheComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "そのまま置ける文はそのまま",
			text: "Mewst export format v1: each post is an article.h-entry.",
			want: "Mewst export format v1: each post is an article.h-entry.",
		},
		{
			// A run of hyphens would end the comment where the text meant to
			// keep going, leaving the rest of the sentence as markup.
			//
			// [Ja] 連続したハイフンは、テキストが続くつもりの場所でコメントを
			// 終わらせ、文の残りをマークアップとして残してしまう。
			name: "コメントを終端させるハイフンの連続は 1 つに詰める",
			text: "format v1 --> alert",
			want: "format v1 - alert",
		},
		{
			name: "3 つ以上続くハイフンも残さない",
			text: "a ---- b",
			want: "a - b",
		},
		{
			// A comment is written raw, so an angle bracket in it could open an
			// element of the archive's own markup.
			//
			// [Ja] コメントは raw で書き出すため、その中の山括弧はアーカイブ自身の
			// マークアップの要素を開いてしまいうる。
			name: "山括弧は取り除く",
			text: "<script>alert(1)</script>",
			want: "scriptalert(1)/script",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := commentText(tt.text); got != tt.want {
				t.Errorf("commentText(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
