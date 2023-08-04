# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  belongs_to :profile

  attribute :viewer_has_stamped, :boolean, default: false

  delegated_type :postable, types: Postable::TYPES, dependent: :destroy

  delegate :comment, to: :postable

  validates :postable_type, inclusion: {in: Postable::TYPES}

  sig { params(posts: ActiveRecord::Relation, viewer: Profile).returns(T::Array[Post]) }
  def self.with_viewer_states(posts:, viewer:)
    stamps = viewer.stamps.where(stampable: posts.map(&:postable))
    stamped_posts = stamps.map(&:post)

    posts.map do |post|
      post.viewer_has_stamped = stamped_posts.include?(post)
      post
    end
  end

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

  sig { returns(Integer) }
  def stamps_count
    if commented_post?
      postable.stamps_count
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
    topic = Mewst::CloudPubsub.client.topic(Rails.configuration.mewst["pubsub_topic_name_add_post_to_home_timeline"])

    followers.find_each do |follower|
      topic.publish_async({profile_id: follower.id, post_id: id}.to_json)
    end
  end

  sig { returns(T::Boolean) }
  def repostable?
    postable_type.in?(Repostable::TYPES)
  end
end
