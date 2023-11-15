# typed: false
# frozen_string_literal: true

RSpec.describe "POST /latest/posts", type: :request, api_version: :latest do
  context "投稿内容が不正なとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`422` を返すこと" do
      post("/latest/posts", headers:, params: {
        content: "a" * (Post::MAXIMUM_CONTENT_LENGTH + 1)
      })
      expect(response).to have_http_status(:unprocessable_entity)

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "content",
            message: "Content is too long (maximum is #{Post::MAXIMUM_CONTENT_LENGTH} characters)"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(422)
    end
  end

  context "入力データが正しいとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`201` を返すこと" do
      expect(Post.count).to eq(0)

      post("/latest/posts", headers:, params: {
        content: "Hello"
      })
      expect(response).to have_http_status(:created)

      expect(Post.count).to eq(1)
      post = Post.first

      expected = {
        post: build_post_resource(post:, viewer_has_followed: false, viewer_has_stamped: false)
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(201)
    end
  end
end
