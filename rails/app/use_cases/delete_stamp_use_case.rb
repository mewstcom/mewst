# typed: strict
# frozen_string_literal: true

class DeleteStampUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, PostRecord
  end

  sig { params(viewer: ActorRecord, target_post: PostRecord).returns(Result) }
  def call(viewer:, target_post:)
    stamp = viewer.profile_record.not_nil!.stamp_records.find_by(post_record: target_post)

    ApplicationRecord.transaction do
      stamp&.unnotify!
      stamp&.reload&.destroy!
    end

    Result.new(post: target_post.reload)
  end
end
