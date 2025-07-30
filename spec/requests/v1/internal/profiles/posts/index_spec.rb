# typed: false
# frozen_string_literal: true

RSpec.xdescribe "GET /v1/internal/@:atname/posts", type: :request do
  context "正常系" do
    let!(:token) { ActionController::HttpAuthentication::Token.encode_credentials(Rails.configuration.mewst["internal_api_token"]) }
    let!(:headers) { {"HTTP_AUTHORIZATION" => token} }
    let!(:actor) { create(:actor, :with_access_token_for_web) }
    let!(:profile) { actor.profile_record }
    let!(:form) { V1::PostForm.new(viewer: actor, content: "Hello") }
    let!(:post) { CreatePostUseCase.new.call(viewer: actor, content: form.content.not_nil!).post }

    it "プロフィールと投稿一覧が返ること" do
      get("/v1/internal/@#{profile.atname}/posts", headers:)

      expect(response).to have_http_status(:ok)

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
