# typed: strict
# frozen_string_literal: true

class FollowProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :target_profile, Profile
  end

  sig { params(viewer: Actor, target_profile: Profile).returns(Result) }
  def call(viewer:, target_profile:)
    follow = viewer.follows.find_by(target_profile: target_profile)

    if follow
      return Result.new(target_profile: follow.target_profile)
    end

    new_follow = viewer.follows.new(target_profile: target_profile, followed_at: Time.current)

    ApplicationRecord.transaction do
      new_follow.save!
      new_follow.notify!
      new_follow.check_suggested!
    end

    Result.new(target_profile: new_follow.target_profile)
  end
end
