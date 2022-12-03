# typed: strict
# frozen_string_literal: true

class PhoneNumberForm < ApplicationForm
  attribute :code, :string

  attr_accessor :phone_number_confirmation

  validates :code, presence: true
  validates :phone_number_confirmation, presence: true
  validate :code_equal_to_phone_number_confirmation_code

  private def code_equal_to_phone_number_confirmation_code
    if code != phone_number_confirmation&.code
      errors.add(:code, :mismatch)
    end
  end
end
