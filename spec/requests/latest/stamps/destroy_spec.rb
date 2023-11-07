# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /latest/posts/:post_id/stamp", type: :request, api_version: :latest do
  context "入力データが正しいとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:target_actor) { create(:actor) }
    let!(:target_profile) { target_actor.profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:post_form) { Latest::PostForm.new(viewer: target_actor, content: "hello") }
    let!(:post) { CreatePostUseCase.new.call(viewer: target_actor, content: post_form.content.not_nil!).post }
    let!(:stamp_form) { Latest::StampForm.new(viewer:, target_post_id: post.id) }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    before do
      CreateStampUseCase.new.call(
        viewer:,
        target_post: stamp_form.target_post.not_nil!
      )
    end

    it "`200` を返すこと" do
      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(1)

      delete("/latest/posts/#{post.id}/stamp", headers:)
      expect(response).to have_http_status(:ok)

      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(0)

      expected = {
        post: {
          id: post.id,
          content: post.content,
          profile: build_profile_resource(profile: target_profile.reload, viewer_has_followed: false),
          published_at: post.published_at.iso8601,
          stamps_count: 0,
          viewer_has_stamped: false
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(200)
    end
  end
end
