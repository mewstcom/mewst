# typed: false
# frozen_string_literal: true

RSpec.describe "GET /@:atname/posts/:post_id", type: :request do
  it "ログインしていないとき、ポストが表示されること" do
    actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: actor.profile)

    get "/@#{actor.atname}/posts/#{post.id}"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include(post.content)
  end

  it "ログインしているとき、ポストが表示されること" do
    viewer = FactoryBot.create(:actor)
    actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: actor.profile)

    sign_in viewer
    get "/@#{actor.atname}/posts/#{post.id}"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include(post.content)
  end

  it "削除されたプロフィールのポストの場合、404エラーが発生すること" do
    actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: actor.profile)
    actor.profile.discard!

    get "/@#{actor.atname}/posts/#{post.id}"

    expect(response).to have_http_status(:not_found)
  end

  it "削除されたポストの場合、404エラーが発生すること" do
    actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: actor.profile)
    post.discard!

    get "/@#{actor.atname}/posts/#{post.id}"

    expect(response).to have_http_status(:not_found)
  end

  it "存在しないatnameの場合、404エラーが発生すること" do
    get "/@nonexistent_user/posts/some_id"

    expect(response).to have_http_status(:not_found)
  end

  it "存在しないポストIDの場合、404エラーが発生すること" do
    actor = FactoryBot.create(:actor)

    get "/@#{actor.atname}/posts/nonexistent_id"

    expect(response).to have_http_status(:not_found)
  end

  it "他のプロフィールのポストを指定した場合、404エラーが発生すること" do
    actor1 = FactoryBot.create(:actor)
    actor2 = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: actor2.profile)

    get "/@#{actor1.atname}/posts/#{post.id}"

    expect(response).to have_http_status(:not_found)
  end

  it "ログインユーザーがポストにスタンプしている場合、正常に表示されること" do
    viewer = FactoryBot.create(:actor)
    actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: actor.profile)
    Stamp.create!(profile: viewer.profile, post: post, stamped_at: Time.current)

    sign_in viewer
    get "/@#{actor.atname}/posts/#{post.id}"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include(post.content)
  end
end
