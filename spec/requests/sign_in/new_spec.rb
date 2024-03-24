# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_in", type: :request do
  context "ログインしているとき" do
    let!(:actor) { FactoryBot.create(:actor) }

    before do
      sign_in actor
    end

    it "ホーム画面にリダイレクトすること" do
      get "/sign_in"

      expect(response).to redirect_to(home_path)
    end
  end

  context "ログインしていないとき" do
    it "ページが表示されること" do
      get "/sign_in"

      expect(response.body).to include("ログイン")
    end
  end
end
