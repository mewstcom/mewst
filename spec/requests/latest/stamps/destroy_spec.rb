# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /latest/posts/:post_id/stamp", type: :request, api_version: :latest do
  context "when valid input data" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:target_actor) { create(:actor) }
    let!(:target_profile) { target_actor.profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:post_form) { Latest::PostForm.new(viewer: target_actor, comment: "hello") }
    let!(:post) { CreatePostUseCase.new.call(viewer: target_actor, comment: post_form.comment.not_nil!).post }
    let!(:stamp_form) { Latest::StampForm.new(viewer:, target_post_id: post.id) }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    before do
      CreateStampUseCase.new.call(
        viewer:,
        target_post: stamp_form.target_post.not_nil!
      )
    end

    it "responses 200" do
      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(1)

      delete("/latest/posts/#{post.id}/stamp", headers:)
      expect(response).to have_http_status(:ok)

      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(0)

      expected = {
        post: {
          id: post.id,
          comment: post.comment,
          profile: {
            atname: target_profile.atname,
            avatar_url: target_profile.avatar_url,
            description: target_profile.description,
            name: target_profile.name,
            viewer_has_followed: false
          },
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
