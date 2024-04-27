# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  extend Enumerize

  include Discard::Model

  ATNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  ATNAME_MIN_LENGTH = 2
  ATNAME_MAX_LENGTH = 20
  MAX_SUGGESTED_FOLLOWS_COUNT = 30 # Note: この数値に深い意味はない

  enumerize :owner_type, in: ProfileOwnerType.values.map(&:serialize)
  enum :avatar_kind, ProfileAvatarKind.values.map { [_1.serialize, _1.serialize] }.to_h, suffix: true

  has_many :actors, dependent: :restrict_with_exception
  has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :inverse_follows, class_name: "Follow", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
  has_many :followees, class_name: "Profile", source: :target_profile, through: :follows
  has_many :followers, class_name: "Profile", source: :source_profile, through: :inverse_follows
  has_many :home_timeline_posts, dependent: :restrict_with_exception
  has_many :notifications, dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
  has_many :posts, dependent: :restrict_with_exception
  has_many :stamps, dependent: :restrict_with_exception
  has_many :suggested_follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :suggested_followees, class_name: "Profile", source: :target_profile, through: :suggested_follows
  has_one :user_profile, dependent: :restrict_with_exception
  has_one :user, through: :user_profile

  scope :sort_by_latest_post, -> { order("last_post_at DESC NULLS LAST") }

  validates :atname,
    format: {with: ATNAME_FORMAT},
    length: {in: ATNAME_MIN_LENGTH..ATNAME_MAX_LENGTH},
    presence: true,
    uniqueness: true,
    unreserved_atname: true

  delegate :delete_post, to: :postability

  sig { returns(ProfileAvatarKind) }
  def deserialized_avatar_kind
    @deserialized_avatar_kind ||= ProfileAvatarKind.deserialize(avatar_kind)
  end

  sig { returns(String) }
  def avatar_url
    case deserialized_avatar_kind
    when ProfileAvatarKind::Default
      ActionController::Base.helpers.asset_url("avatar.png")
    when ProfileAvatarKind::Gravatar
      gravatar_url
    when ProfileAvatarKind::External
      image_url
    else
      T.absurd(deserialized_avatar_kind)
    end.not_nil!
  end

  sig { returns(User) }
  def owner
    profile_owner_type = ProfileOwnerType.deserialize(owner_type.value)

    case profile_owner_type
    when ProfileOwnerType::User
      user
    else
      T.absurd(profile_owner_type)
    end.not_nil!
  end

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

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def following?(target_profile:)
    follows.exists?(target_profile:)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def me?(target_profile:)
    atname == target_profile.atname
  end

  # おすすめプロフィール (自分がフォローしている人がフォローしている人と、その人がフォローしている人) を作成する
  sig { void }
  def create_suggested_follows!
    # おすすめプロフィールを作りすぎないように制限する
    return if suggested_follows.not_checked.size >= MAX_SUGGESTED_FOLLOWS_COUNT

    # Note: 各limitに指定している数値に深い意味はない
    followees.kept.sort_by_latest_post.limit(10).each do |source_followee|
      source_followee.followees.kept.sort_by_latest_post.limit(10).each do |target_followee|
        # 自分がフォローしている人がフォローしている人をおすすめする
        create_suggested_follow!(target_profile: target_followee)

        target_followee.followees.kept.sort_by_latest_post.limit(10).each do |child_target_followee|
          # 自分がフォローしている人がフォローしている人がフォローしている人をおすすめする
          create_suggested_follow!(target_profile: child_target_followee)
        end
      end
    end
  end

  sig { params(target_profile: Profile).void }
  private def create_suggested_follow!(target_profile:)
    return if me?(target_profile:)
    return if following?(target_profile:)

    suggested_follows.where(target_profile:).first_or_create!

    true
  end

  sig { returns(Profile::Postability) }
  private def postability
    Profile::Postability.new(profile: self)
  end
end
