# typed: strict
# frozen_string_literal: true

class CheckSuggestedFollowUseCase < ApplicationUseCase
  sig { params(source_profile: Profile, target_profile: Profile).void }
  def call(source_profile:, target_profile:)
    suggested_follow = source_profile.suggested_follows.find_by(target_profile:)
    return unless suggested_follow

    suggested_follow.touch(:checked_at)

    nil
  end
end
