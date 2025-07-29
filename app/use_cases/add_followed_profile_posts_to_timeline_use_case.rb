# typed: strict
# frozen_string_literal: true

class AddFollowedProfilePostsToTimelineUseCase < ApplicationUseCase
  sig { params(source_profile: ProfileRecord, target_profile: ProfileRecord).void }
  def call(source_profile:, target_profile:)
    target_posts = target_profile.posts.kept.order(published_at: :desc).limit(30)

    ActiveRecord::Base.transaction do
      target_posts.each do |post|
        source_profile.home_timeline.add_post!(post:)
      end
    end

    nil
  end
end
