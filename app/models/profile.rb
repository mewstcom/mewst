# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  include SoftDeletable

  IDNAME_FORMAT = /\A[A-Za-z0-9_]+\z/

  has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id

  delegated_type :profilable, types: %w[User Organization]

  validates :idname, format: {with: IDNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true
end
