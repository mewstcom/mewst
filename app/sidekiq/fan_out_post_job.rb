# typed: strict
# frozen_string_literal: true

class FanOutPostJob
  extend T::Sig

  include Sidekiq::Job

  sig { params(post_id: String).returns(T::Boolean) }
  def perform(post_id)
    post = Post.find(post_id)

    add_post_to_own_home(post:)
    add_post_to_followers_home(post:)

    true
  rescue ActiveRecord::RecordNotFound
    true
  end

  sig { params(post: Post).void }
  private def add_post_to_own_home(post:)
    AddPostToHomeJob.perform_async(post.id, post.profile_id)
  end

  sig { params(post: Post).void }
  private def add_post_to_followers_home(post:)
    followers = post.profile.followers

    followers.find_each do |follower|
      AddPostToHomeJob.perform_async(post.id, follower.id)
    end
  end
end
