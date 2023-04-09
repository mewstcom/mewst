# typed: strict
# frozen_string_literal: true

class Entry < ApplicationRecord
  belongs_to :profile

  delegated_type :entryable, types: [
    Post.to_s
  ], dependent: :destroy

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(String) }
  def timeline_score
    published_at.strftime("%s%L")
  end

  # sig { returns(ActiveSupport::TimeWithZone) }
  # def created_at!
  #   T.cast(created_at, ActiveSupport::TimeWithZone)
  # end

  sig { void }
  def add_to_followers_home_timeline
    followers = profile!.followers
    topic = Mewst::CloudPubsub.client.topic(ENV.fetch("MEWST_PUBSUB_TOPIC_NAME_ADD_ENTRY_TO_HOME_TIMELINE"))

    followers.find_each do |follower|
      topic.publish_async({profile_id: follower.id, entry_id: id}.to_json)
    end
  end
end
