# typed: strict
# frozen_string_literal: true

class PhoneNumberVerificationChallenge
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes

  sig { returns(T.nilable(PhoneNumberVerification)) }
  attr_accessor :phone_number_verification

  attribute :challenged_confirmation_code, :string

  delegate :account, to: :phone_number_verification!

  validates :challenged_confirmation_code, comparison: {equal_to: :confirmation_code}

  sig { returns(PhoneNumberVerification) }
  def phone_number_verification!
    T.cast(phone_number_verification, PhoneNumberVerification)
  end

  private

  delegate :confirmation_code, to: :phone_number_verification!
end
