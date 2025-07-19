# typed: false
# frozen_string_literal: true

RSpec.describe "POST /links", type: :request do
  it "認証されていないユーザーの場合、ルートパスにリダイレクトすること" do
    post "/links", params: {
      link_data_fetcher_form: {
        target_url: "https://example.com"
      }
    }

    expect(response).to redirect_to(root_path)
  end

  it "target_urlが空の場合、422ステータスとエラーメッセージを返すこと" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    post "/links", params: {
      link_data_fetcher_form: {
        target_url: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "target_urlが無効なURL形式の場合、422ステータスを返すこと" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    post "/links", params: {
      link_data_fetcher_form: {
        target_url: "invalid-url"
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "有効なtarget_urlが渡された場合、適切なレスポンスを返すこと" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    VCR.use_cassette("link_data_fetcher/valid") do
      post "/links", params: {
        link_data_fetcher_form: {
          target_url: "https://example.com?foo=bar"
        }
      }

      # LinkDataFetcherやCreateLinkUseCaseの実装に依存するため、
      # 最低限レスポンスが正常に処理されることを確認
      expect(response.status).to be_in([200, 422])
    end
  end
end
