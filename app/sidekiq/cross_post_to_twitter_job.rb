# typed: strict
# frozen_string_literal: true

class CrossPostToTwitterJob
  extend T::Sig

  include Sidekiq::Job

  sig { params(post_id: String).returns(T::Boolean) }
  def perform(post_id)
    post = Post.find(post_id)
    twitter_account = post.profile.twitter_account

    return true unless twitter_account&.can_cross_post?

    twitter_account.tweet(text: post.tweet_text)

    true
  rescue ActiveRecord::RecordNotFound
    true
  end
end
