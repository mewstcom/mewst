# typed: strict
# frozen_string_literal: true

class Services::CreateRepost < Services::Base
  class Result < T::Struct
    const :post, Post
  end

  sig { params(form: Forms::Repost).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    post = form.profile!.posts.create!(kind: :repost, published_at: Time.current)
    post.create_repost!(
      original_follow: form.original_follow!,
      target_post: form.target_post!,
      target_profile: form.target_post!.profile!,
      original_post: form.original_post,
      original_profile: form.original_post.profile!
    )

    form.profile!.home_timeline.add_post(post:)
    FanoutPostJob.perform_async(post_id: post.id)

    Result.new(post:)
  end

  private

  sig { returns(Forms::Repost) }
  attr_reader :form
end
