# typed: strict
# frozen_string_literal: true

class DeletePostJob < ApplicationJob
  queue_with_priority PRIORITY.fetch(:low)

  sig { params(target_post_id: T::Mewst::DatabaseId).void }
  def perform(target_post_id:)
    form = DeletePostForm.new(target_post_id: target_post_id)

    if form.valid?
      DeletePostUseCase.new.call(target_post: form.target_post.not_nil!)
    end
  end
end
