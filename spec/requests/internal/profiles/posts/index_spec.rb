# typed: false
# frozen_string_literal: true

RSpec.describe "GET /internal/@:atname/posts", type: :request, api_version: :internal do
  context "when success" do
    let!(:profile) { create(:user, :with_access_token_for_web).profile }
    let!(:form) { Latest::PostForm.new(comment: "Hello", profile:) }
    let!(:post) { CreatePostUseCase.new.call(profile:, comment: form.comment.not_nil!).post }

    it "returns posts with profile" do
      get("/internal/@#{profile.atname}/posts")

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expected = {
        profile: {
          atname: profile.atname,
          avatar_url: profile.avatar_url,
          description: profile.description,
          name: profile.name,
          viewer_has_followed: false
        },
        posts: [
          {
            id: post.id,
            comment: "Hello",
            profile: {
              atname: profile.atname,
              avatar_url: profile.avatar_url,
              name: profile.name,
              description: profile.description,
              viewer_has_followed: false
            },
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
