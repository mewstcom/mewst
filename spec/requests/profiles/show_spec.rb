# typed: false
# frozen_string_literal: true

RSpec.describe "GET /@:atname", type: :request do
  it "ログインしていないとき、プロフィールページが表示されること" do
    actor = FactoryBot.create(:actor)
    get "/@#{actor.atname}"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("@#{actor.atname}")
  end

  it "ログインしているとき、プロフィール編集リンクが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in actor
    get "/@#{actor.atname}"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("プロフィール編集")
  end

  it "存在しないユーザーの場合、404エラーが発生すること" do
    get "/@nonexistent_user"
    expect(response).to have_http_status(:not_found)
  end
end
