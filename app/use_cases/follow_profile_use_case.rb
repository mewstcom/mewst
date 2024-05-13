# typed: strict
# frozen_string_literal: true

class FollowProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :target_profile, Profile
  end

  sig { params(source_profile: Profile, target_profile: Profile).returns(Result) }
  def call(source_profile:, target_profile:)
    followee = source_profile.followees.find_by(atname: target_profile.atname)

    if followee
      return Result.new(target_profile: followee)
    end

    follow = source_profile.follows.new(target_profile: target_profile, followed_at: Time.current)

    ApplicationRecord.transaction do
      follow.save!
      follow.check_suggested!
      AddFollowedProfilePostsToTimelineJob.perform_later(
        source_profile_id: source_profile.id.not_nil!,
        target_profile_id: target_profile.id.not_nil!
      )
      CreateSuggestedFollowsJob.perform_later(source_profile_id: source_profile.id.not_nil!)
    end

    Result.new(target_profile: follow.target_profile.not_nil!)
  end
end
