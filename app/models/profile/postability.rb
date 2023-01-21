# typed: strict
# frozen_string_literal: true

class Profile::Postability
  extend T::Sig

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { params(content: T.nilable(String), cross_post_to_twitter: T::Boolean).returns(Post) }
  def create_post(content:, cross_post_to_twitter:)
    post = profile.posts.create!(content:)
    profile.twitter_account&.update!(cross_post: cross_post_to_twitter)

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
      profile.home_timeline.remove_post(post:)
    end
  end

  private

  sig { returns(Profile) }
  attr_reader :profile
end
