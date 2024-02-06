# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  extend Enumerize

  include Discard::Model

  include ModelConcerns::TimelineOwnable

  ATNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  ATNAME_MIN_LENGTH = 2
  ATNAME_MAX_LENGTH = 20

  enumerize :profileable_type, in: ProfileableType.values.map(&:serialize)

  has_many :actors, dependent: :restrict_with_exception
  has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :inverse_follows, class_name: "Follow", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
  has_many :followees, class_name: "Profile", source: :target_profile, through: :follows
  has_many :followers, class_name: "Profile", source: :source_profile, through: :inverse_follows
  has_many :notifications, dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
  has_many :posts, dependent: :restrict_with_exception
  has_many :stamps, dependent: :restrict_with_exception
  has_many :suggested_follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :suggested_followees, class_name: "Profile", source: :target_profile, through: :suggested_follows
  has_one :user, dependent: :restrict_with_exception

  scope :sort_by_latest_post, -> { order("last_post_at DESC NULLS LAST") }

  validates :atname,
    format: {with: ATNAME_FORMAT},
    length: {in: ATNAME_MIN_LENGTH..ATNAME_MAX_LENGTH},
    presence: true,
    uniqueness: true,
    unreserved_atname: true

  delegate :delete_post, to: :postability

  sig { returns(Post::PrivateRelation) }
  def followee_posts
    Post.joins(:profile).merge(followees)
  end

  sig { returns(ActiveRecord::Relation) }
  def checkable_suggested_followees
    suggested_followees.kept.merge(SuggestedFollow.not_checked)
  end

  sig { returns(String) }
  def name_or_atname
    name.presence || atname
  end

  sig { returns(Profile::HomeTimeline) }
  def home_timeline
    Profile::HomeTimeline.new(profile: self)
  end

  sig { override.returns(String) }
  def timeline_key
    "timeline:profile:#{id}"
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def following?(target_profile:)
    follows.exists?(target_profile:)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def me?(target_profile:)
    atname == target_profile.atname
  end

  # おすすめプロフィール (自分がフォローしている人がフォローしている人) を作成する
  sig { void }
  def create_suggested_follows!
    # おすすめプロフィールを作りすぎないように制限する
    return if suggested_follows.not_checked.size >= 30 # Note: この数値に深い意味はない

    # Note: 各limitに指定している数値に深い意味はない
    followees.kept.sort_by_latest_post.limit(10).each do |source_followee|
      source_followee.followees.kept.sort_by_latest_post.limit(10).each do |target_profile|
        if !me?(target_profile:) && !following?(target_profile:)
          suggested_follows.where(target_profile:).first_or_create!
        end
      end
    end
  end

  sig { returns(Profile::Postability) }
  private def postability
    Profile::Postability.new(profile: self)
  end
end
