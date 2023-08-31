# typed: strict
# frozen_string_literal: true

class FanoutPostService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :post_id, T::Mewst::DatabaseId
  end

  class Result < T::Struct
    const :post, Post
  end

  sig { params(input: Input).returns(Result) }
  def call(input:)
    post = Post.find(input.post_id)
    followers = post.profile.followers

    batch = GoodJob::Batch.new
    batch.add do
      followers.find_each do |follower|
        AddPostToTimelineJob.perform_later(profile_id: follower.id, post_id: post.id)
      end
    end
    batch.enqueue

    Result.new(post:)
  end
end
