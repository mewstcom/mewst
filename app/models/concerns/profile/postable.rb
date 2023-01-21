# typed: strict
# frozen_string_literal: true

module Profile::Postable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    has_many :posts, dependent: :restrict_with_exception
  end

  sig { params(content: T.nilable(String), cross_post_to_twitter: T::Boolean).returns(Post) }
  def create_post(content:, cross_post_to_twitter:)
    post = posts.create!(content:)
    twitter_account&.update!(cross_post: cross_post_to_twitter)

    if cross_post_to_twitter
      CrossPostToTwitterJob.perform_async(post.id)
    end

    FanOutPostJob.perform_async(post.id)

    post
  end

  sig { params(post: Post).void }
  def delete_post(post:)
    ActiveRecord::Base.transaction do
      post.destroy!
      home_timeline.remove_post(post:)
    end
  end
end
