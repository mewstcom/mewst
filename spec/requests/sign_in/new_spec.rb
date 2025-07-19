# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_in", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in actor

    get "/sign_in"

    expect(response).to redirect_to(home_path)
  end

  it "ログインしていないとき、ログインページが表示されること" do
    get "/sign_in"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("ログイン")
  end

  it "ログインしているとき、既にログイン済みのメッセージが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in actor

    get "/sign_in"

    expect(flash[:notice]).to eq("すでにログインしています")
  end

  it "ログインしていないとき、正しいコンテンツタイプでレスポンスされること" do
    get "/sign_in"

    expect(response.content_type).to match(%r{text/html})
  end

  it "ロケールが設定されていること" do
    get "/sign_in", headers: {"Accept-Language" => "en"}

    expect(response).to have_http_status(:ok)
  end
end
