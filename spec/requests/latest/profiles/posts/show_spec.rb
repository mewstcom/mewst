# typed: false
# frozen_string_literal: true

RSpec.describe "GET /latest/@:atname/posts/:post_id", type: :request, api_version: :latest do
  context "when success" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
    let!(:post) { create(:post) }

    it "returns a post" do
      get("/latest/@#{post.profile.atname}/posts/#{post.id}", headers:)

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
