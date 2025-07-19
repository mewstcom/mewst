# typed: false
# frozen_string_literal: true

RSpec.describe "GET /new", type: :request do
  it "認証されていないユーザーの場合、ルートパスにリダイレクトすること" do
    get "/new"

    expect(response).to redirect_to(root_path)
  end

  it "認証されたユーザーの場合、正常にページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/new"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("post_form")
  end

  it "認証されたユーザーでcontentパラメータありの場合、フォームに内容が設定されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/new", params: {content: "テスト投稿内容"}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("テスト投稿内容")
  end

  it "認証されたユーザーでwith_frameパラメータがtrueの場合、モーダル内にフォームが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/new", params: {with_frame: true}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include('<turbo-frame id="modal">')
    expect(response.body).to include('<turbo-frame id="post-form">')
  end

  it "認証されたユーザーでwith_frameパラメータがfalseの場合、通常のフォームが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/new", params: {with_frame: false}

    expect(response).to have_http_status(:ok)
    expect(response.body).not_to include('<turbo-frame id="post-form">')
  end
end
