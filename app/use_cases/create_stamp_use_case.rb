# typed: strict
# frozen_string_literal: true

class CreateStampUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, PostRecord
  end

  sig { params(viewer: ActorRecord, target_post: PostRecord).returns(Result) }
  def call(viewer:, target_post:)
    stamp = viewer.stamps.find_by(post: target_post)

    if stamp
      return Result.new(post: stamp.post)
    end

    new_stamp = viewer.stamps.new(post: target_post, stamped_at: Time.current)

    ApplicationRecord.transaction do
      new_stamp.save!
      new_stamp.notify!
    end

    Result.new(post: new_stamp.post.reload)
  end
end
