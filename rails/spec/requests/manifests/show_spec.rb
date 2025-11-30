# typed: false
# frozen_string_literal: true

RSpec.describe "GET /manifest", type: :request do
  it "JSONフォーマットでアクセスしたとき、Webアプリケーションマニフェストを返すこと" do
    get "/manifest.json"

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("application/json")

    json_response = JSON.parse(response.body)
    expect(json_response["name"]).to eq("Mewst")
    expect(json_response["short_name"]).to eq("Mewst")
    expect(json_response["display"]).to eq("standalone")
    expect(json_response["start_url"]).to eq("/home")
    expect(json_response["theme_color"]).to eq("#f6f2eb")
    expect(json_response["background_color"]).to eq("#f6f2eb")
    expect(json_response["scope"]).to eq("/")
    expect(json_response).to have_key("icons")
    expect(json_response).to have_key("shortcuts")
    expect(json_response).to have_key("share_target")
  end

  it "HTMLフォーマットでアクセスしたとき、404エラーを返すこと" do
    get "/manifest"

    expect(response).to have_http_status(:not_found)
    expect(response.body).to eq("Not Found")
  end

  it "XMLフォーマットでアクセスしたとき、404エラーを返すこと" do
    get "/manifest.xml"

    expect(response).to have_http_status(:not_found)
    expect(response.body).to eq("Not Found")
  end

  it "ログインしているとき、JSONフォーマットでアクセスしたとき、Webアプリケーションマニフェストを返すこと" do
    actor = FactoryBot.create(:actor)
    sign_in actor

    get "/manifest.json"

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("application/json")

    json_response = JSON.parse(response.body)
    expect(json_response["name"]).to eq("Mewst")
    expect(json_response["short_name"]).to eq("Mewst")
  end

  it "ログインしているとき、HTMLフォーマットでアクセスしたとき、404エラーを返すこと" do
    actor = FactoryBot.create(:actor)
    sign_in actor

    get "/manifest"

    expect(response).to have_http_status(:not_found)
    expect(response.body).to eq("Not Found")
  end
end
