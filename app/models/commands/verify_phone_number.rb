# typed: strict
# frozen_string_literal: true

class Commands::VerifyPhoneNumber < Commands::Base
  include ActiveModel::Model

  sig { returns(T.nilable(User)) }
  attr_reader :user

  validates :confirmation_code, presence: true
  validate :match_confirmation_code

  sig { params(phone_number_verification_challenge: PhoneNumberVerificationChallenge, confirmation_code: T.nilable(String)).void }
  def initialize(phone_number_verification_challenge:, confirmation_code: nil)
    @phone_number_verification_challenge = phone_number_verification_challenge
    @confirmation_code = confirmation_code
    @user = T.let(nil, T.nilable(User))
  end

  sig { returns(T.self_type) }
  def call
    phone_number = PhoneNumber.find_by!(value: phone_number_verification_challenge.phone_number)
    @user = phone_number.user

    phone_number_verification_challenge.destroy

    self
  end

  sig { returns(User) }
  def user!
    T.cast(user, User)
  end

  private

  sig { returns(PhoneNumberVerificationChallenge) }
  attr_reader :phone_number_verification_challenge

  sig { returns(T.nilable(String)) }
  attr_reader :confirmation_code

  sig { void }
  def match_confirmation_code
    if confirmation_code != phone_number_verification_challenge.confirmation_code
      errors.add(:confirmation_code, :mismatch)
    end
  end
end
