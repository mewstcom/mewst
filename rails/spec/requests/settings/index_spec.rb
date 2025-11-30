# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    get "/settings"

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、設定ページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/settings"

    expect(response).to have_http_status(:ok)
  end
end
