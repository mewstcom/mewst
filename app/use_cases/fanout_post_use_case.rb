# typed: strict
# frozen_string_literal: true

class FanoutPostUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, PostRecord
  end

  sig { params(post: PostRecord).returns(Result) }
  def call(post:)
    followers = post.profile_record.not_nil!.follower_records

    batch = GoodJob::Batch.new
    batch.add do
      followers.find_each do |follower|
        AddPostToTimelineJob.perform_later(profile_id: follower.id.not_nil!, post_id: post.id.not_nil!)
      end
    end
    batch.enqueue

    Result.new(post:)
  end
end
