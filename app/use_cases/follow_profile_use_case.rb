# typed: strict
# frozen_string_literal: true

class FollowProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :target_profile, Profile
  end

  sig { params(profile: Profile, target_profile: Profile).returns(Result) }
  def call(profile:, target_profile:)
    followee = profile.followees.find_by(atname: target_profile.atname)

    if followee
      return Result.new(target_profile: followee)
    end

    follow = profile.follows.new(target_profile: target_profile, followed_at: Time.current)

    ApplicationRecord.transaction do
      follow.save!
      follow.check_suggested!
    end

    Result.new(target_profile: follow.target_profile.not_nil!)
  end
end
