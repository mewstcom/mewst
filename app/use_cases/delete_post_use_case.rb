# typed: strict
# frozen_string_literal: true

class DeletePostUseCase < ApplicationUseCase
  sig { params(post_id: T::Mewst::DatabaseId).void }
  def call(post_id:)
    post = Post.find_by(id: post_id)

    return if post.nil?

    post.stamps.find_each(&:destroy!)
    post.destroy!

    nil
  end
end
