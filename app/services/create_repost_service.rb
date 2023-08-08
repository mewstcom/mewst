# typed: strict
# frozen_string_literal: true

class CreateRepostService < ApplicationService
  class Result < T::Struct
    const :post, Post
  end

  sig { params(form: Forms::Repost).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    post = form.viewer!.posts.create!(kind: :repost, published_at: Time.current)
    post.create_repost!(
      comment_post: form.target_post!.comment_post!,
      profile: form.viewer!,
      follow: form.follow!
    )

    form.viewer!.home_timeline.add_post(post:)
    FanoutPostJob.perform_async(post_id: post.id)

    Result.new(post:)
  end

  sig { returns(Forms::Repost) }
  attr_reader :form
  private :form
end
