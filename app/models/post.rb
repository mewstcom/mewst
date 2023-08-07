# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  extend Enumerize

  belongs_to :profile

  has_one :comment_post, dependent: :restrict_with_exception
  has_one :repost, dependent: :restrict_with_exception

  enumerize :kind, in: %i[comment_post repost], predicates: {prefix: true}

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(CommentPost) }
  def comment_post!
    T.must(comment_post)
  end

  sig { returns(Repost) }
  def repost!
    T.must(repost)
  end

  sig { returns(Post) }
  def original_post
    return self if kind_comment_post?

    T.cast(repost!.original_post, Post)
  end

  sig { returns(Integer) }
  def reposts_count
    original_post.comment_post!.reposts_count
  end

  sig { returns(Integer) }
  def stamps_count
    original_post.comment_post!.stamps_count
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
