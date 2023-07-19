# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  belongs_to :profile

  delegated_type :postable, types: Postable::TYPES, dependent: :destroy

  delegate :comment, to: :postable

  validates :postable_type, inclusion: {in: Postable::TYPES}

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(Integer) }
  def reposts_count
    if commented_post?
      postable.reposts_count
    else
      fail
    end
  end

  sig { returns(String) }
  def timeline_score
    published_at.strftime("%s%L")
  end

  sig { void }
  def add_to_followers_home_timeline
    followers = profile!.followers
    topic = Mewst::CloudPubsub.client.topic(ENV.fetch("MEWST_PUBSUB_TOPIC_NAME_ADD_POST_TO_HOME_TIMELINE"))

    followers.find_each do |follower|
      topic.publish_async({profile_id: follower.id, post_id: id}.to_json)
    end
  end

  sig { returns(T::Boolean) }
  def repostable?
    postable_type.in?(Repostable::TYPES)
  end
end
