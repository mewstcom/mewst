# typed: false
# frozen_string_literal: true

RSpec.describe "GET /internal/posts/:post_id", type: :request, api_version: :internal do
  context "ポストが存在しないとき" do
    it "`404` を返すこと" do
      get("/internal/posts/unknown_post_id")

      expect(response).to have_http_status(:not_found)
      assert_response_schema_confirm(404)

      expected = {
        errors: [
          {
            code: "not_found",
            message: "Not found"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)
    end
  end

  context "リクエストが正常なとき" do
    let!(:post) { create(:post) }

    it "ポストを返すこと" do
      get("/internal/posts/#{post.id}")

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expected = {
        post: build_post_resource(post:, viewer_has_followed: false, viewer_has_stamped: false)
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)
    end
  end
end
