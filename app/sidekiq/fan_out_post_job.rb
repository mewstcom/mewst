# typed: strict
# frozen_string_literal: true

class FanOutPostJob
  extend T::Sig

  include Sidekiq::Job

  sig { params(post_id: String).returns(T::Boolean) }
  def perform(post_id)
    post = Post.find(post_id)

    add_post_to_own_home_timeline(post:)
    add_post_to_followers_home_timeline(post:)

    true
  rescue ActiveRecord::RecordNotFound
    true
  end

  private

  sig { params(post: Post).void }
  def add_post_to_own_home_timeline(post:)
    AddPostToHomeTimelineJob.perform_async(post.profile_id, post.id)
  end

  private

  sig { params(post: Post).void }
  def add_post_to_followers_home_timeline(post:)
    followers = T.must(post.profile).followers

    followers.find_each do |follower|
      AddPostToHomeTimelineJob.perform_async(follower.id, post.id)
    end
  end
end
