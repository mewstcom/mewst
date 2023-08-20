# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  extend Enumerize

  MAXIMUM_COMMENT_LENGTH = 500

  belongs_to :profile
  has_many :stamps, dependent: :restrict_with_exception

  validates :comment, length: {maximum: MAXIMUM_COMMENT_LENGTH}, presence: true

  sig { returns(String) }
  def timeline_score
    published_at.strftime("%s%L")
  end

  sig { void }
  def add_to_followers_home_timeline
    followers = profile.not_nil!.followers
    topic = Mewst::CloudPubsub.client.topic(Rails.configuration.mewst["pubsub_topic_name_add_post_to_home_timeline"])

    followers.find_each do |follower|
      topic.publish_async({profile_id: follower.id, post_id: id}.to_json)
    end
  end
end
