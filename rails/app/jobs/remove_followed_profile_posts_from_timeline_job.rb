# typed: strict
# frozen_string_literal: true

class RemoveFollowedProfilePostsFromTimelineJob < ApplicationJob
  sig { params(source_profile_id: T::Mewst::DatabaseId, target_profile_id: T::Mewst::DatabaseId).void }
  def perform(source_profile_id:, target_profile_id:)
    source_profile = ProfileRecord.kept.find(source_profile_id)
    target_profile = ProfileRecord.kept.find(target_profile_id)

    RemoveFollowedProfilePostsFromTimelineUseCase.new.call(source_profile:, target_profile:)
  end
end
