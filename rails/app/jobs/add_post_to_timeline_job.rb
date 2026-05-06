# typed: strict
# frozen_string_literal: true

class AddPostToTimelineJob < ApplicationJob
  queue_with_priority PRIORITY.fetch(:high)

  sig { params(profile_id: T::Mewst::DatabaseId, post_id: T::Mewst::DatabaseId).void }
  def perform(profile_id:, post_id:)
    profile = ProfileRecord.kept.find(profile_id)
    post = PostRecord.find(post_id)

    profile.home_timeline.add_post!(post:)
  end
end
