# typed: false
# frozen_string_literal: true

RSpec.describe "GET /latest/timeline", type: :request, api_version: :latest do
  context "when success" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:profile) { viewer.profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
    let!(:form) { Latest::PostForm.new(viewer:, comment: "Hello") }
    let!(:post) { CreatePostUseCase.new.call(viewer:, comment: form.comment.not_nil!).post }

    it "returns posts on timeline" do
      get("/latest/timeline", headers:)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expected = {
        posts: [
          {
            id: post.id,
            comment: "Hello",
            profile: build_profile_resource(profile:, viewer_has_followed: false),
            published_at: post.published_at.iso8601,
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
