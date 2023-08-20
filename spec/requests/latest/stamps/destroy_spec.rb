# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /latest/posts/:post_id/stamp", type: :request, api_version: :latest do
  context "when valid input data" do
    let!(:profile_1) { create(:profile, :for_user, :with_access_token_for_web) }
    let!(:profile_2) { create(:profile, :for_user) }
    let!(:oauth_access_token) { profile_1.oauth_access_tokens.first }
    let!(:post_form) { Latest::PostForm.new(profile: profile_2, comment: "hello") }
    let!(:post_input) { CreatePostService::Input.from_latest_form(form: post_form) }
    let!(:post) { CreatePostService.new.call(input: post_input).post }
    let!(:stamp_form) { Latest::StampForm.new(profile: profile_1, target_post_id: post.id) }
    let!(:stamp_input) { CreateStampService::Input.from_latest_form(form: stamp_form) }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    before do
      CreateStampService.new.call(input: stamp_input)
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
            atname: profile_2.atname,
            avatar_url: profile_2.avatar_url,
            name: profile_2.name
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
