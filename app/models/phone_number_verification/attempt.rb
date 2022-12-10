# typed: strict
# frozen_string_literal: true

class PhoneNumberVerification::Attempt
  extend T::Sig

  include ActiveModel::Model

  validates :confirmation_code, presence: true
  validate :match_confirmation_code

  sig { params(phone_number_verification: PhoneNumberVerification, confirmation_code: T.nilable(String)).void }
  def initialize(phone_number_verification:, confirmation_code:)
    @phone_number_verification = phone_number_verification
    @confirmation_code = confirmation_code
  end

  private

  sig { returns(PhoneNumberVerification) }
  attr_reader :phone_number_verification

  sig { returns(T.nilable(String)) }
  attr_reader :confirmation_code

  sig { void }
  def match_confirmation_code
    if confirmation_code != phone_number_verification.confirmation_code
      errors.add(:confirmation_code, :mismatch)
    end
  end
end
