# typed: strict
# frozen_string_literal: true

class SignUpForm < ApplicationForm
  attribute :idname, :string

  attr_accessor :phone_number

  validates :idname, format: {with: Profile::IDNAME_FORMAT}, length: {maximum: 20}, presence: true
  validates :phone_number, presence: true
  validate :idname_uniqueness

  sig { returns(T.untyped) }
  private def idname_uniqueness
    if Profile.find_by(idname:)
      errors.add(:idname, :idname_uniqueness)
    end
  end
end
