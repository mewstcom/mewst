# typed: false
# frozen_string_literal: true

RSpec.describe "POST /latest/@:atname/commented_posts", type: :request, api_version: :latest do
  context "when invalid comment" do
    let!(:user) { create(:user, :with_profile, :with_mewst_web_access_token) }
    let!(:profile) { user.profile }
    let!(:oauth_access_token) { user.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "responses 422" do
      post("/latest/@#{profile.atname}/commented_posts", headers:, params: {
        comment: "a" * 501
      })
      expect(response).to have_http_status(:unprocessable_entity)

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "comment",
            message: "Comment is too long (maximum is 500 characters)"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(422)
    end
  end

  context "when valid input data" do
    let!(:user) { create(:user, :with_profile, :with_mewst_web_access_token) }
    let!(:profile) { user.profile }
    let!(:oauth_access_token) { user.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "responses 201" do
      expect(Post.count).to eq(0)
      expect(CommentedPost.count).to eq(0)

      post("/latest/@#{profile.atname}/commented_posts", headers:, params: {
        comment: "Hello"
      })
      expect(response).to have_http_status(:created)

      expect(Post.count).to eq(1)
      expect(CommentedPost.count).to eq(1)
      post = Post.first

      expected = {
        post: {
          id: post.id,
          postable: {
            comment: "Hello"
          },
          postable_type: "commented_post",
          profile: {
            atname: profile.atname,
            avatar_url: profile.avatar_url,
            name: profile.name
          },
          published_at: post.published_at.iso8601(3),
          reposts_count: 0
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(201)
    end
  end
end
