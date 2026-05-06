# typed: false
# frozen_string_literal: true

RSpec.describe "GET /links/new", type: :request do
  it "認証されていないユーザーの場合、ルートパスにリダイレクトすること" do
    get "/links/new", params: {url: "https://example.com"}

    expect(response).to redirect_to(root_path)
  end

  it "認証されたユーザーで有効なurl パラメータありの場合、正常にページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/links/new", params: {url: "https://example.com/path"}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("example.com/path")
    expect(response.body).to include('<input autocomplete="off" type="hidden" value="https://example.com/path" name="link_data_fetcher_form[target_url]" id="link_data_fetcher_form_target_url" />')
  end

  it "認証されたユーザーで無効なurl パラメータありの場合、正常にページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/links/new", params: {url: "invalid-url"}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include('<input autocomplete="off" type="hidden" value="invalid-url" name="link_data_fetcher_form[target_url]" id="link_data_fetcher_form_target_url" />')
  end

  it "認証されたユーザーで空のurl パラメータの場合、正常にページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/links/new", params: {url: ""}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include('<input autocomplete="off" type="hidden" value="" name="link_data_fetcher_form[target_url]" id="link_data_fetcher_form_target_url" />')
  end
end
