# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  extend Enumerize

  include Profile::Followable
  include Profile::Postable
  include Profile::TwitterConnectable
  include SoftDeletable
  include TimelineOwnable
  T.unsafe(self).include ImageUploader::Attachment(:avatar)

  IDNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  PROFILABLE_TYPE_ACCOUNT = :account
  PROFILABLE_TYPE_ORGANIZATION = :organization

  has_one :twitter_account, dependent: :restrict_with_exception

  enumerize :profilable_type, in: [PROFILABLE_TYPE_ACCOUNT, PROFILABLE_TYPE_ORGANIZATION]

  validates :atname, format: {with: IDNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true

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
end
