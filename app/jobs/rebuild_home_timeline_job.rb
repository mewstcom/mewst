# typed: strict
# frozen_string_literal: true

class RebuildHomeTimelineJob < ApplicationJob
  queue_with_priority PRIORITY.fetch(:medium)

  sig { params(profile_id: T::Mewst::DatabaseId, posts_limit: Integer).void }
  def perform(profile_id:, posts_limit:)
    RebuildHomeTimelineUseCase.new.call(profile_id:, posts_limit:)
  end
end
