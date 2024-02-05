# typed: strict
# frozen_string_literal: true

class CreateStampUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(current_actor: Actor, target_post: Post).returns(Result) }
  def call(current_actor:, target_post:)
    stamp = current_actor.stamps.find_by(post: target_post)

    if stamp
      return Result.new(post: stamp.post)
    end

    new_stamp = current_actor.stamps.new(post: target_post, stamped_at: Time.current)

    ApplicationRecord.transaction do
      new_stamp.save!
      new_stamp.notify!
    end

    Result.new(post: new_stamp.post.reload)
  end
end
