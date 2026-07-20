# typed: false
# frozen_string_literal: true

RSpec.describe "GET /home", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    get "/home"

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、ホームタイムラインが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/home"

    expect(response).to have_http_status(:ok)
  end

  it "ログインしているとき、ナビバーの新規投稿リンクがモーダルではなくフルページの /new を指すこと" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/home"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include('href="/new"')
    expect(response.body).not_to include('data-turbo-frame="modal"')
    expect(response.body).not_to include("with_frame")
  end
end
