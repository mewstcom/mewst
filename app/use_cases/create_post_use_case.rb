# typed: strict
# frozen_string_literal: true

class CreatePostUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(current_actor: Actor, content: String, oauth_application: OauthApplication).returns(Result) }
  def call(current_actor:, content:, oauth_application: OauthApplication.mewst_web)
    post = current_actor.posts.new(
      content:,
      published_at: Time.current,
      oauth_application:
    )

    ActiveRecord::Base.transaction do
      post.save!
      current_actor.update_last_post_time!(time: post.published_at)

      current_actor.home_timeline.add_post(post:)
      FanoutPostJob.perform_later(post_id: post.id)
    end

    Result.new(post:)
  end
end
