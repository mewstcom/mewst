# typed: strict
# frozen_string_literal: true

class Profile::Postability
  extend T::Sig

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { params(content: T.nilable(String)).returns(Post) }
  def create_post(content:)
    post = profile.posts.create!(content:)

    profile.home_timeline.add_post(post:)
    FanoutPostJob.perform_async(post_id: post.id)

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
