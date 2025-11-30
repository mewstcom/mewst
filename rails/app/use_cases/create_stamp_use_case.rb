# typed: strict
# frozen_string_literal: true

class CreateStampUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, PostRecord
  end

  sig { params(viewer: ActorRecord, target_post: PostRecord).returns(Result) }
  def call(viewer:, target_post:)
    stamp = viewer.profile_record.not_nil!.stamp_records.find_by(post_record: target_post)

    if stamp
      return Result.new(post: stamp.post_record.not_nil!)
    end

    new_stamp = viewer.profile_record.not_nil!.stamp_records.new(post_record: target_post, stamped_at: Time.current)

    ApplicationRecord.transaction do
      new_stamp.save!
      new_stamp.notify!
    end

    Result.new(post: new_stamp.post_record.not_nil!.reload)
  end
end
