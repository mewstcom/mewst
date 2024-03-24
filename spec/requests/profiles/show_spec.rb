# typed: false
# frozen_string_literal: true

RSpec.describe "GET /@:atname", type: :request do
  context "ログインしていないとき" do
    let!(:actor) { FactoryBot.create(:actor) }

    it "ページが表示されること" do
      get "/@#{actor.atname}"

      expect(response.body).to include("@#{actor.atname}")
    end
  end

  context "ログインしているとき" do
    let!(:actor) { FactoryBot.create(:actor) }

    before do
      sign_in actor
    end

    it "ページが表示されること" do
      get "/@#{actor.atname}"

      expect(response.body).to include("プロフィール編集")
    end
  end
end
