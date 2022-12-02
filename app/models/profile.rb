# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  include SoftDeletable

  IDNAME_FORMAT = /\A[A-Za-z0-9_]+\z/

  has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :posts, dependent: :restrict_with_exception

  enum :profilable_type, {user: 0, organization: 1}, prefix: :as

  validates :idname, format: {with: IDNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true
end
