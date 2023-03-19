# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  extend Enumerize

  include SoftDeletable
  include TimelineOwnable
  T.unsafe(self).include ImageUploader::Attachment(:avatar)

  IDNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  PROFILABLE_TYPE_ACCOUNT = :account
  PROFILABLE_TYPE_ORGANIZATION = :organization

  has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :inverse_follows, class_name: "Follow", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
  has_many :followees, class_name: "Profile", source: :target_profile, through: :follows
  has_many :followers, class_name: "Profile", source: :source_profile, through: :inverse_follows
  has_many :posts, dependent: :restrict_with_exception

  enumerize :profilable_type, in: [PROFILABLE_TYPE_ACCOUNT, PROFILABLE_TYPE_ORGANIZATION]

  validates :atname, format: {with: IDNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true

  delegate :follow, :unfollow, to: :followability
  delegate :create_post, :delete_post, to: :postability
  delegate :upsert_twitter_account, to: :twitter_connectability

  sig { returns(Profile::HomeTimeline) }
  def home_timeline
    Profile::HomeTimeline.new(profile: self)
  end

  sig { override.returns(String) }
  def timeline_key
    "timeline:profile:#{id}"
  end

  sig { returns(T.nilable(ImageUploader::UploadedFile)) }
  def master_avatar
    avatar&.fetch(:master, nil)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def following?(target_profile:)
    follows.exists?(target_profile:)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def me?(target_profile:)
    atname == target_profile.atname
  end

  private

  sig { returns(Profile::Followability) }
  def followability
    Profile::Followability.new(source_profile: self)
  end

  sig { returns(Profile::Postability) }
  def postability
    Profile::Postability.new(profile: self)
  end

  sig { returns(Profile::TwitterConnectability) }
  def twitter_connectability
    Profile::TwitterConnectability.new(profile: self)
  end
end
