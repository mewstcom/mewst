# typed: false
# frozen_string_literal: true

# rubocop:disable RSpec/EmptyExampleGroup
RSpec.xdescribe "GET /v1/internal/posts/:post_id", type: :request do
  # context "ポストが存在しないとき" do
  #   let!(:token) { ActionController::HttpAuthentication::Token.encode_credentials(Rails.configuration.mewst["internal_api_token"]) }
  #   let!(:headers) { {"HTTP_AUTHORIZATION" => token} }

  #   it "`404` を返すこと" do
  #     get("/v1/internal/posts/unknown_post_id", headers:)

  #     expect(response).to have_http_status(:not_found)

  #     expected = {
  #       errors: [
  #         {
  #           code: "not_found",
  #           message: "Not found"
  #         }
  #       ]
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)
  #   end
  # end

  # context "リクエストが正常なとき" do
  #   let!(:token) { ActionController::HttpAuthentication::Token.encode_credentials(Rails.configuration.mewst["internal_api_token"]) }
  #   let!(:headers) { {"HTTP_AUTHORIZATION" => token} }
  #   let!(:post) { create(:post_record) }

  #   it "ポストを返すこと" do
  #     get("/v1/internal/posts/#{post.id}", headers:)

  #     expect(response).to have_http_status(:ok)

  #     expected = {
  #       post: build_post_resource(post:, viewer_has_followed: false, viewer_has_stamped: false)
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)
  #   end
  # end
end
# rubocop:enable RSpec/EmptyExampleGroup
