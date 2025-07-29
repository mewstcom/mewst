# typed: strict
# frozen_string_literal: true

class UnfollowProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :target_profile, ProfileRecord
  end

  sig { params(profile: ProfileRecord, target_profile: ProfileRecord).returns(Result) }
  def call(profile:, target_profile:)
    follow = profile.follow_records.find_by(target_profile_id: target_profile.id)

    ApplicationRecord.transaction do
      follow&.destroy!
      RemoveFollowedProfilePostsFromTimelineJob.perform_later(
        source_profile_id: profile.id.not_nil!,
        target_profile_id: target_profile.id.not_nil!
      )
    end

    Result.new(target_profile: target_profile.reload)
  end
end
