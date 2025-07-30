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

  it "ログインしているとき、他のユーザーのプロフィールページではプロフィール編集リンクが表示されないこと" do
    actor = FactoryBot.create(:actor)
    other_actor = FactoryBot.create(:actor)
    sign_in actor
    get "/@#{other_actor.atname}"

    expect(response).to have_http_status(:ok)
    expect(response.body).not_to include("プロフィール編集")
  end

  it "削除されたプロフィールの場合、404エラーが発生すること" do
    actor = FactoryBot.create(:actor)
    actor.profile_record.discard!
    get "/@#{actor.atname}"
    expect(response).to have_http_status(:not_found)
  end

  it "正常なページネーションパラメータを受け取れること" do
    actor = FactoryBot.create(:actor)
    FactoryBot.create(:post_record, profile_record: actor.profile_record)
    FactoryBot.create(:post_record, profile_record: actor.profile_record)

    get "/@#{actor.atname}"
    expect(response).to have_http_status(:ok)
  end

  it "無効なカーソルの場合、プロフィールページにリダイレクトされること" do
    actor = FactoryBot.create(:actor)

    get "/@#{actor.atname}", params: {after: "invalid_cursor"}

    expect(response).to have_http_status(:moved_permanently)
    expect(response.location).to include("/@#{actor.atname}")
  end

  it "存在しないユーザーの場合、404エラーが発生すること" do
    get "/@nonexistent_user"
    expect(response).to have_http_status(:not_found)
  end
end
