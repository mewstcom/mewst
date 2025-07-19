# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/email", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    get "/settings/email"

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、メール設定ページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/settings/email"

    expect(response).to have_http_status(:ok)
  end
end
