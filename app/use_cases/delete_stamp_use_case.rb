# typed: strict
# frozen_string_literal: true

class DeleteStampUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(current_actor: Actor, target_post: Post).returns(Result) }
  def call(current_actor:, target_post:)
    stamp = current_actor.stamps.find_by(post: target_post)

    ApplicationRecord.transaction do
      stamp&.unnotify!
      stamp&.destroy!
    end

    Result.new(post: target_post.reload)
  end
end
