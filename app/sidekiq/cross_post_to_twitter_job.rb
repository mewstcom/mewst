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

    twitter_account.tweet(text: tweet_text(post:))

    true
  rescue ActiveRecord::RecordNotFound
    true
  end

  private

  sig { params(post: Post).returns(String) }
  def tweet_text(post:)
    if post.content.length < 130
      return post.content
    end

    mewst_url = Rails.application.routes.url_helpers.post_url(post.profile!.atname, post.id, host: ENV.fetch("MEWST_HOST"))

    [
      post.content.truncate(130),
      "Read more: #{mewst_url}"
    ].join("\n")
  end
end
