# typed: strict
# frozen_string_literal: true

class CreatePostUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(viewer: Actor, content: String).returns(Result) }
  def call(viewer:, content:)
    post = viewer.posts.new(
      content:,
      published_at: Time.current,
      oauth_application: OauthApplication.mewst_web # TODO: Web API公開のタイミングで別のアプリを指定できるようにする
    )

    ActiveRecord::Base.transaction do
      post.save!
      viewer.profile.not_nil!.update!(last_post_at: post.published_at)

      viewer.home_timeline.add_post(post:)
      FanoutPostJob.perform_later(post_id: post.id)
    end

    Result.new(post:)
  end
end
