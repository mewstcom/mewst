# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/profile", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    get "/settings/profile"

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、プロフィール設定ページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/settings/profile"

    expect(response).to have_http_status(:ok)
  end

  it "ログインしているとき、現在のプロフィール情報がフォームに設定されること" do
    actor = FactoryBot.create(:actor)
    profile = actor.profile_record
    profile.update!(
      atname: "testuser",
      name: "テストユーザー",
      description: "これはテストです",
      avatar_kind: "gravatar",
      gravatar_email: "test@example.com"
    )
    sign_in(actor)

    get "/settings/profile"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("testuser")
    expect(response.body).to include("テストユーザー")
    expect(response.body).to include("これはテストです")
    expect(response.body).to include("test@example.com")
  end

  it "ログインしているとき、外部画像URLを使用している場合、その情報がフォームに設定されること" do
    actor = FactoryBot.create(:actor)
    profile = actor.profile_record
    profile.update!(
      atname: "imageuser",
      name: "画像ユーザー",
      avatar_kind: "external",
      image_url: "https://example.com/avatar.png"
    )
    sign_in(actor)

    get "/settings/profile"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("imageuser")
    expect(response.body).to include("画像ユーザー")
    expect(response.body).to include("https://example.com/avatar.png")
  end
end
