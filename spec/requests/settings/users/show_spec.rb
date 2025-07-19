# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/user", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    get "/settings/user"

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、ユーザー設定ページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/settings/user"

    expect(response).to have_http_status(:ok)
  end

  it "ログインしているとき、現在のロケールとタイムゾーンがフォームに設定されること" do
    actor = FactoryBot.create(:actor)
    user = actor.user
    user.update!(
      locale: "ja",
      time_zone: "Asia/Tokyo"
    )
    sign_in(actor)

    get "/settings/user"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("ja")
    expect(response.body).to include("Asia/Tokyo")
  end

  it "ログインしているとき、英語ロケールとニューヨークのタイムゾーンが正しくフォームに設定されること" do
    actor = FactoryBot.create(:actor)
    user = actor.user
    user.update!(
      locale: "en",
      time_zone: "America/New_York"
    )
    sign_in(actor)

    get "/settings/user"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("en")
    expect(response.body).to include("America/New_York")
  end
end
