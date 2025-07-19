# typed: false
# frozen_string_literal: true

RSpec.describe "GET /@:atname/atom", type: :request do
  it "ログインしていないとき、ATOMフィードが表示されること" do
    actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, actor:)
    get "/@#{actor.atname}/atom"

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("application/atom+xml")
    expect(response.body).to include(actor.atname)
    expect(response.body).to include(post.content)
  end

  it "ログインしているとき、ATOMフィードが表示されること" do
    actor = FactoryBot.create(:actor)
    other_actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, actor:)
    sign_in other_actor
    get "/@#{actor.atname}/atom"

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("application/atom+xml")
    expect(response.body).to include(actor.atname)
    expect(response.body).to include(post.content)
  end

  it "存在しないユーザーの場合、404エラーが発生すること" do
    get "/@nonexistent_user/atom"
    expect(response).to have_http_status(:not_found)
  end
end