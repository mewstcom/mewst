# typed: strict
# frozen_string_literal: true

class CreateCommentPostService < ApplicationService
  class Input < T::Struct
    const :profile, Profile
    const :comment, String

    sig { params(form: Latest::CommentPostForm).returns(Input) }
    def self.from_latest_form(form:)
      new(
        profile: form.profile!,
        comment: form.comment
      )
    end
  end

  class Result < T::Struct
    const :post, Post
  end

  sig { params(input: Input).returns(Result) }
  def call
    post = input.profile.posts.create!(kind: :comment_post, published_at: Time.current)
    post.create_comment_post!(comment: input.comment)

    input.profile.home_timeline.add_post(post:)
    FanoutPostJob.perform_async(post_id: post.id)

    Result.new(post:)
  end
end
