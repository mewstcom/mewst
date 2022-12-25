# typed: strict
# frozen_string_literal: true

class PhoneNumberVerification < ApplicationRecord
  validates :phone_number, presence: true
  validates :raw_phone_number, presence: true
  validates :confirmation_code, format: {with: /\A\d{6}\z/}, presence: true

  sig { returns(String) }
  def self.generate_confirmation_code
    6.times.map { rand(10) }.join
  end

  sig { returns(T.nilable(Account)) }
  def account
    @account ||= T.let(Account.find_by(phone_number:), T.nilable(Account))
  end
end
