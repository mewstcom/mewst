# typed: false
# frozen_string_literal: true

RSpec.xdescribe "DELETE /v1/posts/:post_id", type: :request, api_version: :v1 do
  # context "リクエストデータが不正なとき" do
  #   let!(:viewer) { create(:actor, :with_access_token_for_web) }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
  #   let!(:post) { create(:post_record) }

  #   it "`422` を返すこと" do
  #     delete("/v1/posts/#{post.id}", headers:)
  #     expect(response).to have_http_status(:unprocessable_entity)

  #     expected = {
  #       errors: [
  #         {
  #           code: "invalid_input_data",
  #           field: "post_id",
  #           message: "Post not found"
  #         }
  #       ]
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)

  #     assert_response_schema_confirm(422)
  #   end
  # end

  # context "リクエストデータが正しいとき" do
  #   let!(:viewer) { create(:actor, :with_access_token_for_web) }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
  #   let!(:post) { create(:post_record, profile: viewer.profile_record) }

  #   it "`204` を返すこと" do
  #     expect(Post.kept.count).to eq(1)

  #     delete("/v1/posts/#{post.id}", headers:)
  #     expect(response).to have_http_status(:no_content)

  #     expect(Post.kept.count).to eq(0)

  #     assert_response_schema_confirm(204)
  #   end
  # end
end
