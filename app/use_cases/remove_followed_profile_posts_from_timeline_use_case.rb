# typed: strict
# frozen_string_literal: true

class RemoveFollowedProfilePostsFromTimelineUseCase < ApplicationUseCase
  sig { params(source_profile: Profile, target_profile: Profile).void }
  def call(source_profile:, target_profile:)
    source_profile.home_timeline_posts.joins(:post).merge(target_profile.posts).find_each do |home_timeline_post|
      home_timeline_post.destroy!
    end

    nil
  end
end
