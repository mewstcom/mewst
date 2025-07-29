# typed: strict
# frozen_string_literal: true

class FollowProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :target_profile, ProfileRecord
  end

  sig { params(source_profile: ProfileRecord, target_profile: ProfileRecord).returns(Result) }
  def call(source_profile:, target_profile:)
    followee = source_profile.followee_records.find_by(atname: target_profile.atname)

    if followee
      return Result.new(target_profile: followee)
    end

    follow = source_profile.follow_records.new(target_profile_record: target_profile, followed_at: Time.current)

    ApplicationRecord.transaction do
      follow.save!
      follow.check_suggested!
      AddFollowedProfilePostsToTimelineJob.perform_later(
        source_profile_id: source_profile.id.not_nil!,
        target_profile_id: target_profile.id.not_nil!
      )
      CreateSuggestedFollowsJob.perform_later(source_profile_id: source_profile.id.not_nil!)
    end

    Result.new(target_profile: follow.target_profile_record.not_nil!)
  end
end
