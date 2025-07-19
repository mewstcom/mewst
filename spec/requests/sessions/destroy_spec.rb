# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /sign_out", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    delete("/sign_out")

    expect(response).to redirect_to(root_path)
  end

  it "ログインしているとき、ルートページにリダイレクトすること" do
    user = FactoryBot.create(:user, email: "test@example.com", password: "password")
    actor = FactoryBot.create(:actor, user:)

    # ログイン
    sign_in(actor, password: "password")

    delete("/sign_out")

    expect(response).to redirect_to(root_path)
    expect(flash[:notice]).to eq("ログアウトしました")
  end

  it "ログインしているとき、セッションクッキーが削除されること" do
    user = FactoryBot.create(:user, email: "test@example.com", password: "password")
    actor = FactoryBot.create(:actor, user:)

    # ログイン
    sign_in(actor, password: "password")
    expect(cookies[Session::COOKIE_KEY]).not_to be_nil

    delete("/sign_out")

    expect(cookies[Session::COOKIE_KEY]).to eq("")
  end

  it "ログアウト後、認証が必要なページにアクセスするとルートページにリダイレクトすること" do
    user = FactoryBot.create(:user, email: "test@example.com", password: "password")
    actor = FactoryBot.create(:actor, user:)

    # ログイン
    sign_in(actor, password: "password")

    # ログアウト
    delete("/sign_out")

    # 認証が必要なページにアクセス
    get(home_path)
    expect(response).to redirect_to(root_path)
  end
end
