# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  include SoftDeletable
  include Inboxable

  IDNAME_FORMAT = /\A[A-Za-z0-9_]+\z/

  has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :inverse_follows, class_name: "Follow", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
  has_many :followees, class_name: "Profile", source: :target_profile, through: :follows
  has_many :followers, class_name: "Profile", source: :source_profile, through: :inverse_follows
  has_many :posts, dependent: :restrict_with_exception

  enum :profilable_type, {user: 0, organization: 1}, prefix: :as

  validates :idname, format: {with: IDNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true

  sig { override.returns(String) }
  def inbox_key
    "inbox:profile:#{id}"
  end
end
