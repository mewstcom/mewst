# typed: false
# frozen_string_literal: true

RSpec.xdescribe "POST /v1/posts/:post_id/stamp", type: :request, api_version: :v1 do
  context "ポストIDが不正なとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`422` を返すこと" do
      post("/v1/posts/invalid_post_id/stamp", headers:)
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
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:post_form) { V1::PostForm.new(viewer: target_actor, content: "hello") }
    let!(:target_post) { CreatePostUseCase.new.call(viewer: target_actor, content: post_form.content.not_nil!).post }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`204` を返すこと" do
      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(0)

      post("/v1/posts/#{target_post.id}/stamp", headers:)
      expect(response).to have_http_status(:created)

      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(1)

      expected = {
        post: build_post_resource(post: target_post.reload, viewer_has_followed: false, viewer_has_stamped: true)
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(201)
    end
  end
end
