# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  IDNAME_FORMAT = /\A[A-Za-z0-9_]+\z/

  delegated_type :profilable, types: %w[User Organization]

  validates :idname, format: {with: IDNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true
end
