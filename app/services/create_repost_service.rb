# typed: strict
# frozen_string_literal: true

class CreateRepostService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :target_post, Post
    const :viewer, Profile
    const :follow, Follow

    sig { params(form: Latest::RepostForm).returns(Input) }
    def self.from_latest_form(form:)
      new(
        target_post: form.target_post.not_nil!,
        viewer: form.viewer.not_nil!,
        follow: form.follow.not_nil!
      )
    end
  end

  class Result < T::Struct
    const :post, Post
  end

  sig { params(input: Input).returns(Result) }
  def call(input:)
    post = input.viewer.posts.create!(kind: :repost, published_at: Time.current)
    post.create_repost!(
      target_comment_post: input.target_post.comment_post.not_nil!,
      profile: input.viewer,
      follow: input.follow
    )

    input.viewer.home_timeline.add_post(post:)
    FanoutPostJob.perform_async(post_id: post.id)

    Result.new(post:)
  end
end
