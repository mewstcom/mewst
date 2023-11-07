# typed: false
# frozen_string_literal: true

RSpec.describe "POST /latest/posts/:post_id/stamp", type: :request, api_version: :latest do
  context "ポストIDが不正なとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`422` を返すこと" do
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

  context "入力データが正しいとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:target_actor) { create(:actor) }
    let!(:target_profile) { target_actor.profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:post_form) { Latest::PostForm.new(viewer: target_actor, content: "hello") }
    let!(:target_post) { CreatePostUseCase.new.call(viewer: target_actor, content: post_form.content.not_nil!).post }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`204` を返すこと" do
      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(0)

      post("/latest/posts/#{target_post.id}/stamp", headers:)
      expect(response).to have_http_status(:created)

      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(1)

      expected = {
        post: {
          id: target_post.id,
          content: target_post.content,
          profile: build_profile_resource(profile: target_profile, viewer_has_followed: false),
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
