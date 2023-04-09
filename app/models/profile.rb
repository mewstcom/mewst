# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  extend Enumerize

  include SoftDeletable
  include TimelineOwnable

  IDNAME_FORMAT = /\A[A-Za-z0-9_]+\z/

  has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :inverse_follows, class_name: "Follow", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
  has_many :followees, class_name: "Profile", source: :target_profile, through: :follows
  has_many :followers, class_name: "Profile", source: :source_profile, through: :inverse_follows
  has_many :entries, dependent: :restrict_with_exception

  validates :atname, format: {with: IDNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true

  delegate :follow, :unfollow, to: :followability
  delegate :create_post_entry, :delete_entry, to: :entryability

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

  private

  sig { returns(Profile::Followability) }
  def followability
    Profile::Followability.new(source_profile: self)
  end

  sig { returns(Profile::Entryability) }
  def entryability
    Profile::Entryability.new(profile: self)
  end
end
