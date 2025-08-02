# typed: strict
# frozen_string_literal: true

class ProfileRecord < ApplicationRecord
  self.table_name = "profiles"

  extend Enumerize

  include Discard::Model

  ATNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  ATNAME_MIN_LENGTH = 2
  ATNAME_MAX_LENGTH = 20
  MAX_SUGGESTED_FOLLOWS_COUNT = 30 # Note: この数値に深い意味はない

  enumerize :owner_type, in: ProfileOwnerType.values.map(&:serialize)
  enum :avatar_kind, ProfileAvatarKind.values.map { [_1.serialize, _1.serialize] }.to_h, suffix: true

  has_many :actor_records, class_name: "ActorRecord", dependent: :restrict_with_exception, foreign_key: :profile_id
  has_many :follow_records, class_name: "FollowRecord", dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile_record
  has_many :inverse_follow_records, class_name: "FollowRecord", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile_record
  has_many :followee_records, class_name: "ProfileRecord", source: :target_profile_record, through: :follow_records
  has_many :follower_records, class_name: "ProfileRecord", source: :source_profile_record, through: :inverse_follow_records
  has_many :home_timeline_post_records, class_name: "HomeTimelinePostRecord", dependent: :restrict_with_exception, foreign_key: :profile_id
  has_many :notification_records, class_name: "NotificationRecord", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile_record
  has_many :post_records, class_name: "PostRecord", dependent: :restrict_with_exception, foreign_key: :profile_id
  has_many :stamp_records, class_name: "StampRecord", dependent: :restrict_with_exception, foreign_key: :profile_id
  has_many :suggested_follow_records, class_name: "SuggestedFollowRecord", dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile_record
  has_many :suggested_followee_records, class_name: "ProfileRecord", source: :target_profile_record, through: :suggested_follow_records
  has_one :user_profile_record, class_name: "UserProfileRecord", dependent: :restrict_with_exception, foreign_key: :profile_id
  has_one :user_record, class_name: "UserRecord", through: :user_profile_record

  scope :sort_by_latest_post, -> { order("last_post_at DESC NULLS LAST") }
  scope :search_by_keywords, ->(q:) {
    words = q.split
    conditions = words
      .map.with_index { |_, i| "(atname ILIKE :word#{i} OR name ILIKE :word#{i} OR description ILIKE :word#{i})" }
      .join(" AND ")
    parameters = words.map.with_index { |word, i| {"word#{i}": "%#{word}%"} }.reduce({}, :merge)
    where(conditions, parameters)
  }

  validates :atname,
    format: {with: ATNAME_FORMAT},
    length: {in: ATNAME_MIN_LENGTH..ATNAME_MAX_LENGTH},
    presence: true,
    uniqueness: true,
    unreserved_atname: true

  delegate :delete_post, to: :postability

  sig { returns(ProfileAvatarKind) }
  def deserialized_avatar_kind
    ProfileAvatarKind.deserialize(avatar_kind)
  end

  sig { returns(String) }
  def avatar_url
    kind = deserialized_avatar_kind

    case kind
    when ProfileAvatarKind::Default
      ActionController::Base.helpers.asset_url("avatar.png")
    when ProfileAvatarKind::Gravatar
      gravatar_url
    when ProfileAvatarKind::External
      image_url
    else
      T.absurd(kind)
    end.not_nil!
  end

  sig { returns(UserRecord) }
  def owner
    profile_owner_type = ProfileOwnerType.deserialize(owner_type.value)

    case profile_owner_type
    when ProfileOwnerType::User
      user_record
    else
      T.absurd(profile_owner_type)
    end.not_nil!
  end

  sig { returns(PostRecord::PrivateRelation) }
  def followee_posts
    PostRecord.joins(:profile_record).merge(followee_records)
  end

  sig { returns(ActiveRecord::Relation) }
  def checkable_suggested_followees
    suggested_followee_records.kept.merge(SuggestedFollowRecord.not_checked)
  end

  sig { returns(String) }
  def name_or_atname
    name.presence || atname
  end

  sig { returns(ProfileRecord::HomeTimeline) }
  def home_timeline
    ProfileRecord::HomeTimeline.new(profile: self)
  end

  sig { params(target_profile: ProfileRecord).returns(T::Boolean) }
  def following?(target_profile:)
    follow_records.exists?(target_profile_id: target_profile.id)
  end

  sig { params(target_profile: ProfileRecord).returns(T::Boolean) }
  def me?(target_profile:)
    atname == target_profile.atname
  end

  # おすすめプロフィール (自分がフォローしている人がフォローしている人と、その人がフォローしている人) を作成する
  sig { void }
  def create_suggested_follows!
    # おすすめプロフィールを作りすぎないように制限する
    return if suggested_follow_records.not_checked.size >= MAX_SUGGESTED_FOLLOWS_COUNT

    # Note: 各limitに指定している数値に深い意味はない
    followee_records.kept.sort_by_latest_post.limit(10).each do |source_followee|
      source_followee.followee_records.kept.sort_by_latest_post.limit(10).each do |target_followee|
        # 自分がフォローしている人がフォローしている人をおすすめする
        create_suggested_follow!(target_profile: target_followee)

        target_followee.followee_records.kept.sort_by_latest_post.limit(10).each do |child_target_followee|
          # 自分がフォローしている人がフォローしている人がフォローしている人をおすすめする
          create_suggested_follow!(target_profile: child_target_followee)
        end
      end
    end
  end

  sig { params(target_profile: ProfileRecord).void }
  private def create_suggested_follow!(target_profile:)
    return if me?(target_profile:)
    return if following?(target_profile:)

    suggested_follow_records.where(target_profile_id: target_profile.id).first_or_create!

    true
  end

  sig { returns(ProfileRecord::Postability) }
  private def postability
    ProfileRecord::Postability.new(profile: self)
  end
end
