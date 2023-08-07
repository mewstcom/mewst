# typed: false
# frozen_string_literal: true

RSpec.describe "GET /latest/timeline", type: :request, api_version: :latest do
  context "when success" do
    let!(:profile) { create(:profile, :for_user, :with_access_token_for_web) }
    let!(:oauth_access_token) { profile.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
    let!(:form) { Forms::CommentPost.new(comment: "Hello", profile:) }
    let!(:post) { Services::CreateCommentPost.new(form:).call.post }

    it "returns posts on timeline" do
      get("/latest/timeline", headers:)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expected = {
        posts: [
          {
            id: post.id,
            kind: "comment_post",
            postable: {
              comment: "Hello"
            },
            profile: {
              atname: profile.atname,
              avatar_url: profile.avatar_url,
              name: profile.name
            },
            published_at: post.published_at.iso8601(3),
            reposts_count: 0,
            stamps_count: 0,
            viewer_has_stamped: false
          }
        ],
        page_info: {
          has_next_page: false,
          has_previous_page: false,
          start_cursor: nil,
          end_cursor: post.id
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)
    end
  end
end
