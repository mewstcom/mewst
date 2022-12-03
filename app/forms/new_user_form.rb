# typed: strict
# frozen_string_literal: true

class NewUserForm < ApplicationForm
  attribute :idname, :string

  sig { returns(T.nilable(PhoneNumberConfirmation)) }
  attr_accessor :phone_number_confirmation

  validates :idname, format: {with: Profile::IDNAME_FORMAT}, length: {maximum: 20}, presence: true
  validates :phone_number_confirmation, presence: true
  validate :idname_uniqueness

  sig { returns(T.untyped) }
  private def idname_uniqueness
    if Profile.find_by(idname:)
      errors.add(:idname, :idname_uniqueness)
    end
  end
end
