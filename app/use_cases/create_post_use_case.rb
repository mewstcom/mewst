# typed: strict
# frozen_string_literal: true

class CreatePostUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(viewer: Actor, content: String).returns(Result) }
  def call(viewer:, content:)
    post = viewer.posts.new(content:, published_at: Time.current)

    ActiveRecord::Base.transaction do
      post.save!

      viewer.home_timeline.add_post(post:)
      FanoutPostJob.perform_later(post_id: post.id)
    end

    Result.new(post:)
  end
end
