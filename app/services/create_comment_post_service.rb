# typed: strict
# frozen_string_literal: true

class CreateCommentPostService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :profile, Profile
    const :comment, String

    sig { params(form: Latest::CommentPostForm).returns(Input) }
    def self.from_latest_form(form:)
      new(
        profile: form.profile.not_nil!,
        comment: form.comment.not_nil!
      )
    end
  end

  class Result < T::Struct
    const :post, Post
  end

  sig { params(input: Input).returns(Result) }
  def call(input:)
    post = input.profile.posts.create!(kind: :comment_post, published_at: Time.current)
    post.create_comment_post!(comment: input.comment)

    input.profile.home_timeline.add_post(post:)
    FanoutPostJob.perform_async(post_id: post.id)

    Result.new(post:)
  end
end
