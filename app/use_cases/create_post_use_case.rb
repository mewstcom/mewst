# typed: strict
# frozen_string_literal: true

class CreatePostUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(profile: Profile, comment: String).returns(Result) }
  def call(profile:, comment:)
    post = profile.posts.new(comment:, published_at: Time.current)

    ActiveRecord::Base.transaction do
      post.save!

      profile.home_timeline.add_post(post:)
      FanoutPostJob.perform_later(post_id: post.id)
    end

    Result.new(post:)
  end
end
