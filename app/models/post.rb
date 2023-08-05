# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  extend Enumerize

  belongs_to :profile

  has_one :comment_post, dependent: :restrict_with_exception
  has_one :repost, dependent: :restrict_with_exception

  enumerize :kind, in: %i[comment_post repost]

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(String) }
  def timeline_score
    published_at.strftime("%s%L")
  end

  sig { void }
  def add_to_followers_home_timeline
    followers = profile!.followers
    topic = Mewst::CloudPubsub.client.topic(Rails.configuration.mewst["pubsub_topic_name_add_post_to_home_timeline"])

    followers.find_each do |follower|
      topic.publish_async({profile_id: follower.id, post_id: id}.to_json)
    end
  end
end
