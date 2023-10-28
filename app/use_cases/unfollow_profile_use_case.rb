# typed: strict
# frozen_string_literal: true

class UnfollowProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :target_profile, Profile
  end

  sig { params(viewer: Actor, target_profile: Profile).returns(Result) }
  def call(viewer:, target_profile:)
    follow = viewer.follows.find_by(target_profile: target_profile)

    ApplicationRecord.transaction do
      follow&.destroy!
      follow&.unnotify!
    end

    Result.new(target_profile: target_profile.reload)
  end
end
