# typed: false
# frozen_string_literal: true

RSpec.describe "GET /", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in actor

    get "/"

    expect(response).to redirect_to(home_path)
    expect(response).to have_http_status(:found)
  end

  it "ログインしていないとき、ウェルカムページが表示されること" do
    get "/"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("現在ベータ版です")
  end
end
