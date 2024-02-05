# typed: strict
# frozen_string_literal: true

class UnfollowProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :target_profile, Profile
  end

  sig { params(profile: Profile, target_profile: Profile).returns(Result) }
  def call(profile:, target_profile:)
    follow = profile.follows.find_by(target_profile: target_profile)

    ApplicationRecord.transaction do
      follow&.destroy!
    end

    Result.new(target_profile: target_profile.reload)
  end
end
