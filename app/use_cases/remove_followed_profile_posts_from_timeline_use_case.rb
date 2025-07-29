# typed: strict
# frozen_string_literal: true

class RemoveFollowedProfilePostsFromTimelineUseCase < ApplicationUseCase
  sig { params(source_profile: ProfileRecord, target_profile: ProfileRecord).void }
  def call(source_profile:, target_profile:)
    source_profile.home_timeline_post_records.joins(:post_record).merge(target_profile.post_records).find_each do |home_timeline_post|
      home_timeline_post.destroy!
    end

    nil
  end
end
