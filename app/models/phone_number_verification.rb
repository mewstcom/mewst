# typed: strict
# frozen_string_literal: true

class PhoneNumberVerification < ApplicationRecord
  validates :phone_number, presence: true
  validates :phone_number_origin, presence: true
  validates :confirmation_code, presence: true

  sig { params(confirmation_code: T.nilable(String)).returns(PhoneNumberVerification::Attempt) }
  def new_attempt(confirmation_code: nil)
    PhoneNumberVerification::Attempt.new(phone_number_verification: self, confirmation_code:)
  end

  sig { params(idname: T.nilable(String), locale: T.nilable(Symbol)).returns(User::Creator) }
  def new_user_creator(idname: nil, locale: nil)
    User::Creator.new(phone_number_verification: self, idname:, locale:)
  end

  sig { returns(T.self_type) }
  def set_confirmation_code
    self.confirmation_code = 6.times.map { rand(10) }.join
    self
  end

  sig { returns(T.self_type) }
  def send_sms
    save!

    SendPhoneNumberVerificationMessageJob.perform_async(id)

    self
  end
end
