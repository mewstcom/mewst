# typed: false
# frozen_string_literal: true

RSpec.describe "GET /", type: :request do
  context "ログインしているとき" do
    let!(:actor) { FactoryBot.create(:actor) }

    before do
      sign_in actor
    end

    it "ホーム画面にリダイレクトすること" do
      get "/"

      expect(response).to redirect_to(home_path)
    end
  end
end
