# typed: strict
# frozen_string_literal: true

class CreateRepostService < ApplicationService
  class Input < T::Struct
    const :target_post, Post
    const :viewer, Profile
    const :follow, Follow

    sig { params(form: Latest::RepostForm).returns(Input) }
    def self.from_latest_form(form:)
      new(
        target_post: form.target_post!,
        viewer: form.profile!,
        follow: form.follow!
      )
    end
  end

  class Result < T::Struct
    const :post, Post
  end

  sig { params(input: Input).returns(Result) }
  def call
    post = input.viewer.posts.create!(kind: :repost, published_at: Time.current)
    post.create_repost!(
      comment_post: input.target_post.comment_post!,
      profile: input.viewer,
      follow: input.follow
    )

    input.viewer!.home_timeline.add_post(post:)
    FanoutPostJob.perform_async(post_id: post.id)

    Result.new(post:)
  end
end
