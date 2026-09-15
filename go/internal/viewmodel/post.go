package viewmodel

import "github.com/mewstcom/mewst/go/internal/model"

// MaximumPostContentLengthはドメインの上限値をテンプレート向けに再公開する
// (templatesはdepguardによりmodelに直接依存できない)。投稿フォームの文字数
// カウンターがカウンターの初期値として使用する。
const MaximumPostContentLength = model.MaximumPostContentLength
