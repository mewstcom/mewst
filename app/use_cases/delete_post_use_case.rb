# typed: strict
# frozen_string_literal: true

class DeletePostUseCase < ApplicationUseCase
  sig { params(target_post: Post).void }
  def call(target_post:)
    target_post.stamps.find_each(&:destroy!)
    target_post.destroy!

    nil
  end
end
