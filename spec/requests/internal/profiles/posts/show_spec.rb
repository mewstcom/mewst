# typed: false
# frozen_string_literal: true

RSpec.describe "GET /internal/@:atname/posts/:post_id", type: :request, api_version: :internal do
  context "when a request failed" do
    let!(:post) { create(:post) }

    it "responses 404" do
      get("/internal/@#{post.profile.atname}/posts/unknown_post_id")

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

  context "when a request success" do
    let!(:post) { create(:post) }

    it "returns a post" do
      get("/internal/@#{post.profile.atname}/posts/#{post.id}")

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
