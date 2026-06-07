package viewmodel

import "github.com/mewstcom/mewst/go/internal/model"

// MaximumPostContentLength re-exports the domain limit for templates, which
// cannot depend on the model package directly (depguard). The character
// counter on the post form uses it as the counter's starting value.
//
// [Ja] MaximumPostContentLength はドメインの上限値をテンプレート向けに再公開する
// (templates は depguard により model に直接依存できない)。投稿フォームの文字数
// カウンターがカウンターの初期値として使用する。
const MaximumPostContentLength = model.MaximumPostContentLength
