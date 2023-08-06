# typed: false
# frozen_string_literal: true

RSpec.describe "POST /latest/posts/:post_id/reposts", type: :request, api_version: :latest do
  context "when invalid post_id" do
    let!(:profile) { create(:profile, :for_user, :with_access_token_for_web) }
    let!(:oauth_access_token) { profile.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "responses 422" do
      post("/latest/posts/invalid_post_id/reposts", headers:)
      expect(response).to have_http_status(:unprocessable_entity)

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "post_id",
            message: "Post not found"
          },
          {
            code: "invalid_input_data",
            field: nil,
            message: "Repost can not be done because you do not follow the original poster"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(422)
    end
  end

  context "when valid input data" do
    let!(:profile_1) { create(:profile, :for_user, :with_access_token_for_web) }
    let!(:profile_2) { create(:profile, :for_user) }
    let!(:oauth_access_token) { profile_1.oauth_access_tokens.first }
    let!(:comment_post_form) { Forms::CommentPost.new(profile: profile_2, comment: "hello") }
    let!(:target_post) { Services::CreateCommentPost.new(form: comment_post_form).call.post }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    before do
      profile_1.follow(target_profile: profile_2)
    end

    it "responses 201" do
      expect(Post.count).to eq(1)
      expect(Repost.count).to eq(0)

      post("/latest/posts/#{target_post.id}/reposts", headers:)
      expect(response).to have_http_status(:created)

      expect(Post.count).to eq(2)
      expect(CommentPost.count).to eq(1)
      post = Post.where.not(id: target_post.id).first

      expected = {
        post: {
          id: post.id,
          kind: "repost",
          postable: {
            original_post: {
              id: target_post.id,
              kind: "comment_post",
              postable: {
                comment: target_post.comment_post!.comment
              },
              profile: {
                atname: profile_2.atname,
                avatar_url: profile_2.avatar_url,
                name: profile_2.name
              },
              published_at: target_post.published_at.iso8601(3),
              reposts_count: 1
            },
            original_profile: {
              atname: profile_2.atname,
              avatar_url: profile_2.avatar_url,
              name: profile_2.name
            },
            target_post: {
              id: target_post.id,
              kind: "comment_post",
              postable: {
                comment: target_post.comment_post!.comment
              },
              profile: {
                atname: profile_2.atname,
                avatar_url: profile_2.avatar_url,
                name: profile_2.name
              },
              published_at: target_post.published_at.iso8601(3),
              reposts_count: 1
            },
            target_profile: {
              atname: profile_2.atname,
              avatar_url: profile_2.avatar_url,
              name: profile_2.name
            }
          },
          profile: {
            atname: profile_1.atname,
            avatar_url: profile_1.avatar_url,
            name: profile_1.name
          },
          published_at: post.published_at.iso8601(3),
          reposts_count: 1
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(201)
    end
  end
end
