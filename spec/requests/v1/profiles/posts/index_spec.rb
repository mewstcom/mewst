# typed: false
# frozen_string_literal: true

RSpec.xdescribe "GET /v1/@:atname/posts", type: :request, api_version: :v1 do
  context "正常系" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:profile) { viewer.profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
    let!(:form) { V1::PostForm.new(viewer:, content: "Hello") }
    let!(:post) { CreatePostUseCase.new.call(viewer:, content: form.content.not_nil!).post }

    it "プロフィールと投稿一覧が返ること" do
      get("/v1/@#{profile.atname}/posts", headers:)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expected = {
        profile: build_profile_resource(profile:, viewer_has_followed: false),
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
