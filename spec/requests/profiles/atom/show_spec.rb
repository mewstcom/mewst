# typed: false
# frozen_string_literal: true

RSpec.describe "GET /@:atname/atom", type: :request do
  it "ログインしていないとき、ATOMフィードが表示されること" do
    actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: actor.profile)
    get "/@#{actor.atname}/atom"

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("application/atom+xml")
    expect(response.body).to include(actor.atname)
    expect(response.body).to include(post.content)
  end

  it "ログインしているとき、ATOMフィードが表示されること" do
    actor = FactoryBot.create(:actor)
    other_actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: actor.profile)
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

  it "ソフト削除されたプロフィールの場合、404エラーが発生すること" do
    actor = FactoryBot.create(:actor)
    actor.profile.discard
    get "/@#{actor.atname}/atom"
    expect(response).to have_http_status(:not_found)
  end

  it "投稿がないユーザーの場合、空のATOMフィードが表示されること" do
    actor = FactoryBot.create(:actor)
    get "/@#{actor.atname}/atom"

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("application/atom+xml")
    expect(response.body).to include(actor.atname)
  end

  it "ソフト削除された投稿は含まれないこと" do
    actor = FactoryBot.create(:actor)
    FactoryBot.create(:post, profile: actor.profile, content: "通常の投稿")
    post2 = FactoryBot.create(:post, profile: actor.profile, content: "削除される投稿")
    post2.discard
    get "/@#{actor.atname}/atom"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("通常の投稿")
    expect(response.body).not_to include("削除される投稿")
  end

  it "投稿が15件を超える場合、最新の15件のみが含まれること" do
    actor = FactoryBot.create(:actor)
    # 古い投稿を5件作成（これらは表示されないはず）
    5.times do |i|
      FactoryBot.create(:post, profile: actor.profile, content: "古い投稿#{i + 1}", published_at: (20 + i).minutes.ago)
    end

    # 新しい投稿を15件作成（これらが表示されるはず）
    15.times do |i|
      FactoryBot.create(:post, profile: actor.profile, content: "新しい投稿#{i + 1}", published_at: (14 - i).minutes.ago)
    end

    get "/@#{actor.atname}/atom"

    expect(response).to have_http_status(:ok)

    # 新しい投稿15件が含まれることを確認
    (1..15).each do |i|
      expect(response.body).to include("新しい投稿#{i}")
    end

    # 古い投稿5件は含まれないことを確認
    (1..5).each do |i|
      expect(response.body).not_to include("古い投稿#{i}")
    end
  end
end
