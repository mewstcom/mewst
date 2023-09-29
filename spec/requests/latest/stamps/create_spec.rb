# typed: false
# frozen_string_literal: true

RSpec.describe "POST /latest/posts/:post_id/stamp", type: :request, api_version: :latest do
  context "when invalid post_id" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "responses 422" do
      post("/latest/posts/invalid_post_id/stamp", headers:)
      expect(response).to have_http_status(:unprocessable_entity)

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "post_id",
            message: "Post not found"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(422)
    end
  end

  context "when valid input data" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:target_actor) { create(:actor) }
    let!(:target_profile) { target_actor.profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:post_form) { Latest::PostForm.new(viewer: target_actor, comment: "hello") }
    let!(:target_post) { CreatePostUseCase.new.call(viewer: target_actor, comment: post_form.comment.not_nil!).post }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "responses 204" do
      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(0)

      post("/latest/posts/#{target_post.id}/stamp", headers:)
      expect(response).to have_http_status(:created)

      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(1)

      expected = {
        post: {
          id: target_post.id,
          comment: target_post.comment,
          profile: profile_resource(profile: target_profile, viewer_has_followed: false),
          published_at: target_post.published_at.iso8601,
          stamps_count: 1,
          viewer_has_stamped: true
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(201)
    end
  end
end
