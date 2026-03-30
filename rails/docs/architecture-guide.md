# アーキテクチャガイド

このドキュメントは、Rails 版 Mewst のアーキテクチャパターンを説明します。

## アーキテクチャパターン

Rails 版 Mewst は、標準の MVC アーキテクチャに加え、以下のパターンを導入しています：

### Records（ActiveRecord モデル）

データベーステーブルに対応する ActiveRecord モデルを配置します。

- **配置**: `app/records/`
- **命名**: `{Model}Record`（例: `UserRecord`, `PostRecord`）
- **責務**: データの永続化、リレーション定義、基本的なバリデーション

### Use Cases（ユースケース）

ビジネスロジックを担当します。

- **配置**: `app/use_cases/`
- **命名**: `{Action}{Entity}UseCase`（例: `CreatePostUseCase`, `FollowProfileUseCase`）
- **メソッド**: `call` メソッドを実装

### ViewComponent

再利用可能な UI コンポーネントを実装します。

- **配置**: `app/components/`
- **命名**: `{ComponentName}Component`
- **テンプレート**: ERB を使用

## クラス間の依存関係ルール

| クラス     | 依存可能な先                                   |
| ---------- | ---------------------------------------------- |
| Component  | Component, Form, Model                         |
| Controller | Form, Model, Record, Repository, UseCase, View |
| Form       | Record, Validator                              |
| Job        | UseCase                                        |
| Mailer     | Model, Record, Repository, View                |
| Model      | Model                                          |
| Policy     | Record                                         |
| Record     | Record                                         |
| Repository | Model, Record, Policy                          |
| UseCase    | Job, Mailer, Record                            |
| Validator  | Record                                         |
| View       | Component, Form, Model                         |

### UseCaseとJobの依存関係について

UseCaseとJobの間には相互依存が存在しますが、以下のルールで循環依存を回避します：

- **UseCase → Job**: `perform_later`メソッドによるキューへの追加のみ許可
- **Job → UseCase**: ジョブ実行時のUseCase呼び出しは許可
- **重要**: UseCaseからJobインスタンスの直接実行（`perform`メソッド）は禁止

```ruby
# ✅ 良い例：UseCaseからジョブをキューに追加
class Users::CreateUseCase < ApplicationUseCase
  def call
    user = UserRecord.create!(...)
    Users::SendWelcomeEmailJob.perform_later(user.id)  # キューに追加のみ
  end
end

# ❌ 悪い例：UseCaseからジョブを直接実行
class Users::CreateUseCase < ApplicationUseCase
  def call
    user = UserRecord.create!(...)
    Users::SendWelcomeEmailJob.new.perform(user.id)  # 直接実行は禁止
  end
end
```

### UseCaseクラスを使用する場合

- ✅ データベースへの永続化を伴う処理
- ✅ 複数のモデル/レコードにまたがる複雑なビジネスロジックで永続化を伴うもの
- ✅ トランザクション管理が必要な処理

### UseCaseクラスを使用しない場合

- ❌ データベースへの永続化を伴わない処理（URL生成、データ変換など）
- ❌ 単一のモデル/レコードに閉じた処理（モデルやレコードのメソッドとして定義）

### トランザクション処理

**重要**: UseCaseクラスでトランザクションを張る場合は、必ず `#with_transaction` メソッドを使用すること

```ruby
# ✅ 良い例：with_transactionを使用
class Users::CreateUseCase < ApplicationUseCase
  def call
    with_transaction do
      user = UserRecord.create!(...)
      ProfileRecord.create!(user:, ...)
    end
  end
end

# ❌ 悪い例：transactionを直接使用
class Users::CreateUseCase < ApplicationUseCase
  def call
    ApplicationRecord.transaction do
      # with_transactionを使うべき
    end
  end
end
```

**重要**: Controller、Job、Rakeタスク内で永続化処理を書く場合は、必ずUseCaseクラスを定義すること

## 命名規則

- Controller: `(ModelPlural)::(ActionName)Controller`
- UseCase: `(ModelPlural)::(Verb)UseCase`
- Form: `(ModelPlural)::(Noun)Form`
- Repository: `(Model)Repository`
- View: `(ModelPlural)::(ActionName)View`
- Component: `(UIComponentPlural)::(Noun)Component`

## 重要な原則

- ネストしたトランザクションを避ける
- レコードのコールバックを避ける
- View/Componentでのデータベースアクセスを防ぐ
- 問題が解決されるなら、レイヤーを跨いだ依存も許可
- 説明的な命名規則
- コメントは日本語で記載
- 1行100文字以内
