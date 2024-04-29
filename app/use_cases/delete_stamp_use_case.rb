# typed: strict
# frozen_string_literal: true

class DeleteStampUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(viewer: Actor, target_post: Post).returns(Result) }
  def call(viewer:, target_post:)
    stamp = viewer.stamps.find_by(post: target_post)

    ApplicationRecord.transaction do
      stamp&.unnotify!
      stamp&.reload&.destroy!
    end

    Result.new(post: target_post.reload)
  end
end
