# typed: strict
# frozen_string_literal: true

class FanoutPostUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(post_id: T::Mewst::DatabaseId).returns(Result) }
  def call(post_id:)
    post = Post.find(post_id)
    followers = post.profile.not_nil!.followers

    batch = GoodJob::Batch.new
    batch.add do
      followers.find_each do |follower|
        AddPostToTimelineJob.perform_later(profile_id: follower.id, post_id: post.id.not_nil!)
      end
    end
    batch.enqueue

    Result.new(post:)
  end
end
