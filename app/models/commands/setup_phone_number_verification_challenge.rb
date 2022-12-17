# typed: strict
# frozen_string_literal: true

class Commands::SetupPhoneNumberVerificationChallenge < Commands::Base
  include ActiveModel::Model

  sig { returns(T.nilable(PhoneNumberVerificationChallenge)) }
  attr_reader :phone_number_verification_challenge

  validates :phone_number, presence: true
  validates :phone_number_origin, presence: true
  validates :confirmation_code, presence: true

  sig { params(phone_number: T.nilable(String), phone_number_origin: T.nilable(String), confirmation_code: T.nilable(String)).void }
  def initialize(phone_number: nil, phone_number_origin: nil, confirmation_code: nil)
    @phone_number = phone_number
    @phone_number_origin = phone_number_origin
    @confirmation_code = confirmation_code
    @phone_number_verification_challenge = T.let(nil, T.nilable(PhoneNumberVerificationChallenge))
  end

  sig { returns(T.self_type) }
  def call
    @phone_number_verification_challenge = PhoneNumberVerificationChallenge.create!(phone_number:, phone_number_origin:, confirmation_code:)

    SendPhoneNumberVerificationMessageJob.perform_async(@phone_number_verification_challenge.id)

    self
  end

  sig { returns(PhoneNumberVerificationChallenge) }
  def phone_number_verification_challenge!
    T.cast(phone_number_verification_challenge, PhoneNumberVerificationChallenge)
  end

  private

  sig { returns(T.nilable(String)) }
  attr_reader :phone_number, :phone_number_origin, :confirmation_code
end
