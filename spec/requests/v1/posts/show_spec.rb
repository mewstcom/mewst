# typed: false
# frozen_string_literal: true

# rubocop:disable RSpec/EmptyExampleGroup
RSpec.xdescribe "GET /v1/posts/:post_id", type: :request, api_version: :v1 do
  # context "ポストIDが存在しないとき" do
  #   let!(:viewer) { create(:actor, :with_access_token_for_web) }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

  #   it "`404` を返すこと" do
  #     get("/v1/posts/unknown_post_id", headers:)

  #     expect(response).to have_http_status(:not_found)
  #     assert_response_schema_confirm(404)

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
  #   let!(:viewer) { create(:actor, :with_access_token_for_web) }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
  #   let!(:post) { create(:post_record) }

  #   it "ポストを返ること" do
  #     get("/v1/posts/#{post.id}", headers:)

  #     expect(response).to have_http_status(:ok)
  #     assert_response_schema_confirm(200)

  #     expected = {
  #       post: build_post_resource(post:, viewer_has_followed: false, viewer_has_stamped: false)
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)
  #   end
  # end
end
# rubocop:enable RSpec/EmptyExampleGroup
