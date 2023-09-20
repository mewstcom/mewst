# typed: false
# frozen_string_literal: true

RSpec.describe "POST /latest/posts/:post_id/stamp", type: :request, api_version: :latest do
  context "when invalid post_id" do
    let!(:user) { create(:user, :with_access_token_for_web) }
    let!(:profile) { user.profile }
    let!(:oauth_access_token) { profile.oauth_access_tokens.first }
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
    let!(:user_1) { create(:user, :with_access_token_for_web) }
    let!(:user_2) { create(:user) }
    let!(:profile_1) { user_1.profile }
    let!(:profile_2) { user_2.profile }
    let!(:oauth_access_token) { profile_1.oauth_access_tokens.first }
    let!(:post_form) { Latest::PostForm.new(profile: profile_2, comment: "hello") }
    let!(:target_post) { CreatePostUseCase.new.call(profile: form.profile.not_nil!, comment: form.comment.not_nil!).post }
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
          profile: {
            atname: profile_2.atname,
            avatar_url: profile_2.avatar_url,
            description: profile_2.description,
            name: profile_2.name,
            viewer_has_followed: false
          },
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
