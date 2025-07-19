# typed: false
# frozen_string_literal: true

RSpec.describe "GET /password_reset", type: :request do
  it "ログインしていないとき、正常にレスポンスが返されること" do
    get "/password_reset"

    expect(response).to have_http_status(:ok)
  end

  it "ログインしているとき、ホームページにリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/password_reset"

    expect(response).to redirect_to(home_path)
  end
end
