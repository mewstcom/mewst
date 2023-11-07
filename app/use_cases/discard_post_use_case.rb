# typed: strict
# frozen_string_literal: true

class DiscardPostUseCase < ApplicationUseCase
  sig { params(target_post: Post).void }
  def call(target_post:)
    ApplicationRecord.transaction do
      target_post.discard!
      DeletePostJob.perform_later(post_id: target_post.id.not_nil!)
    end

    nil
  end
end
