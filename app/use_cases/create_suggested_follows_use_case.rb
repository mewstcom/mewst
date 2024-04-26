# typed: strict
# frozen_string_literal: true

class CreateSuggestedFollowsUseCase < ApplicationUseCase
  sig { params(source_profile: Profile).void }
  def call(source_profile:)
    source_profile.create_suggested_follows!

    nil
  end
end
