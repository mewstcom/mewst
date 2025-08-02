# typed: false
# frozen_string_literal: true

# rubocop:disable RSpec/EmptyExampleGroup
RSpec.xdescribe "DELETE /v1/posts/:post_id/stamp", type: :request, api_version: :v1 do
  # context "入力データが正しいとき" do
  #   let!(:viewer) { create(:actor, :with_access_token_for_web) }
  #   let!(:target_actor) { create(:actor) }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:post_form) { V1::PostForm.new(viewer: target_actor, content: "hello") }
  #   let!(:post) { CreatePostUseCase.new.call(viewer: target_actor, content: post_form.content.not_nil!).post }
  #   let!(:stamp_form) { V1::StampForm.new(viewer:, target_post_id: post.id) }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

  #   before do
  #     CreateStampUseCase.new.call(
  #       viewer:,
  #       target_post: stamp_form.target_post.not_nil!
  #     )
  #   end

  #   it "`200` を返すこと" do
  #     expect(PostRecord.count).to eq(1)
  #     expect(StampRecord.count).to eq(1)

  #     delete("/v1/posts/#{post.id}/stamp", headers:)
  #     expect(response).to have_http_status(:ok)

  #     expect(PostRecord.count).to eq(1)
  #     expect(StampRecord.count).to eq(0)

  #     expected = {
  #       post: build_post_resource(post:, viewer_has_followed: false, viewer_has_stamped: false)
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)

  #     assert_response_schema_confirm(200)
  #   end
  # end
end
# rubocop:enable RSpec/EmptyExampleGroup
