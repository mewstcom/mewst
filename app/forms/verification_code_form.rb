# typed: strict
# frozen_string_literal: true

class VerificationCodeForm < ApplicationForm
  attribute :verification_code, :string

  sig { returns(T.nilable(PhoneNumberConfirmation)) }
  attr_accessor :phone_number_confirmation

  validates :verification_code, presence: true
  validate :verification_code_equal_to_phone_number_confirmation_verification_code

  private

  sig { void }
  def verification_code_equal_to_phone_number_confirmation_verification_code
    if verification_code != phone_number_confirmation&.verification_code
      errors.add(:verification_code, :mismatch)
    end
  end
end
