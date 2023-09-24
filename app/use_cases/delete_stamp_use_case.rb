# typed: strict
# frozen_string_literal: true

class DeleteStampUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(viewer: Actor, target_post: Post).returns(Result) }
  def call(viewer:, target_post:)
    stamp = viewer.stamps.where(post: target_post).sole
    stamp.destroy!

    Result.new(post: target_post.reload)
  end
end
