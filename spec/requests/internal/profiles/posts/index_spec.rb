# typed: false
# frozen_string_literal: true

RSpec.describe "GET /internal/@:atname/posts", type: :request, api_version: :internal do
  context "正常系" do
    let!(:actor) { create(:actor, :with_access_token_for_web) }
    let!(:profile) { actor.profile }
    let!(:form) { Latest::PostForm.new(viewer: actor, content: "Hello") }
    let!(:post) { CreatePostUseCase.new.call(viewer: actor, content: form.content.not_nil!).post }

    it "プロフィールと投稿一覧が返ること" do
      get("/internal/@#{profile.atname}/posts")

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expected = {
        profile: {
          id: profile.id,
          atname: profile.atname,
          avatar_url: profile.avatar_url,
          description: profile.description,
          name: profile.name,
          viewer_has_followed: false
        },
        posts: [
          build_post_resource(post:, viewer_has_followed: false, viewer_has_stamped: false)
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
