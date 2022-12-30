# typed: strict
# frozen_string_literal: true

class AddPostToHomeTimelineJob
  extend T::Sig

  include Sidekiq::Job

  sig { params(profile_id: String, post_id: String).returns(T::Boolean) }
  def perform(profile_id, post_id)
    profile = Profile.only_kept.find_by(id: profile_id)
    post = Post.find_by(id: post_id)

    if profile && post
      profile.home_timeline.add_post(post:)
    end

    true
  rescue ActiveRecord::RecordNotFound
    true
  end
end
