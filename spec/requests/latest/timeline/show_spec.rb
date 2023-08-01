# typed: false
# frozen_string_literal: true

RSpec.describe "GET /latest/@:atname/timeline", type: :request, api_version: :latest do
  context "when success" do
    let!(:user) { create(:user, :with_profile, :with_mewst_web_access_token) }
    let!(:profile) { user.first_profile }
    let!(:oauth_access_token) { user.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
    let!(:form) { Forms::CommentedPost.new(comment: "Hello", profile:) }
    let!(:post) { Services::CreateCommentedPost.new(form:).call.post }

    it "returns posts on timeline" do
      get("/latest/@#{profile.atname}/timeline", headers:)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expected = {
        posts: [
          {
            id: post.id,
            postable: {
              comment: "Hello"
            },
            postable_type: "commented_post",
            profile: {
              atname: profile.atname,
              avatar_url: profile.avatar_url,
              name: profile.name
            },
            published_at: post.published_at.iso8601(3),
            reposts_count: 0
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
