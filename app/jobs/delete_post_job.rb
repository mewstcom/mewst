# typed: strict
# frozen_string_literal: true

class DeletePostJob < ApplicationJob
  queue_with_priority PRIORITY.fetch(:low)

  sig { params(post_id: T::Mewst::DatabaseId).void }
  def perform(post_id:)
    DeletePostUseCase.new.call(post_id:)
  end
end
