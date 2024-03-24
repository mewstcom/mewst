# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_up", type: :request do
  context "ログインしているとき" do
    let!(:actor) { FactoryBot.create(:actor) }

    before do
      sign_in actor
    end

    it "ホーム画面にリダイレクトすること" do
      get "/sign_up"

      expect(response).to redirect_to(home_path)
    end
  end

  context "ログインしていないとき" do
    it "ページが表示されること" do
      get "/sign_up"

      expect(response.body).to include("アカウント作成")
    end
  end
end
