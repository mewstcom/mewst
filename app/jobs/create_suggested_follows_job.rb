# typed: strict
# frozen_string_literal: true

class CreateSuggestedFollowsJob < ApplicationJob
  sig { params(source_profile_id: T::Mewst::DatabaseId).void }
  def perform(source_profile_id:)
    source_profile = Profile.kept.find(source_profile_id)

    CreateSuggestedFollowsUseCase.new.call(source_profile:)
  end
end
