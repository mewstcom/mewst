# typed: strict
# frozen_string_literal: true

class FollowProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :target_profile, Profile
  end

  sig { params(viewer: Profile, target_profile: Profile).returns(Result) }
  def call(viewer:, target_profile:)
    follow = input.viewer.follows.where(target_profile: input.target_profile).first_or_initialize(followed_at: Time.current)

    follow.save!

    Result.new(target_profile: follow.target_profile)
  end
end
