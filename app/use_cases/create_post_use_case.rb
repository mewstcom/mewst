# typed: strict
# frozen_string_literal: true

class CreatePostUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, PostRecord
  end

  sig { params(viewer: ActorRecord, content: String, canonical_url: String, oauth_application: OauthApplication).returns(Result) }
  def call(viewer:, content:, canonical_url:, oauth_application: OauthApplication.mewst_web)
    post = viewer.posts.new(
      content:,
      published_at: Time.current,
      oauth_application:
    )
    link = canonical_url.present? ? LinkRecord.find_by(canonical_url:) : nil

    ActiveRecord::Base.transaction do
      post.save!
      post.create_post_link!(link:) unless link.nil?
      viewer.update_last_post_time!(time: post.published_at)

      viewer.home_timeline.add_post!(post:)
      FanoutPostJob.perform_later(post_id: post.id)
    end

    Result.new(post:)
  end
end
