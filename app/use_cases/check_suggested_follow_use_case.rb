# typed: strict
# frozen_string_literal: true

class CheckSuggestedFollowUseCase < ApplicationUseCase
  sig { params(viewer: Actor, target_profile: Profile).void }
  def call(viewer:, target_profile:)
    suggested_follow = viewer.suggested_follows.find_by(target_profile:)

    return unless suggested_follow

    suggested_follow.touch(:checked_at)

    nil
  end
end
