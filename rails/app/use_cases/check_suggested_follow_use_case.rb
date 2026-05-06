# typed: strict
# frozen_string_literal: true

class CheckSuggestedFollowUseCase < ApplicationUseCase
  sig { params(source_profile: ProfileRecord, target_profile: ProfileRecord).void }
  def call(source_profile:, target_profile:)
    suggested_follow = source_profile.suggested_follow_records.find_by(target_profile_id: target_profile.id)
    return unless suggested_follow

    suggested_follow.touch(:checked_at)

    nil
  end
end
