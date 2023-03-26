# typed: strict
# frozen_string_literal: true

class Verification < ApplicationRecord
  extend Enumerize

  EVENT_PASSWORD_RESET = :password_reset
  EVENT_SIGN_UP = :sign_up
  EXPIRES_IN = 15.minutes

  enumerize :event, in: [EVENT_PASSWORD_RESET, EVENT_SIGN_UP]

  scope :active, -> { where("created_at > ?", EXPIRES_IN.ago) }
  scope :succeeded, -> { where.not(succeeded_at: nil) }

  validates :email, presence: true
  validates :code, format: {with: /\A\d{6}\z/}, presence: true

  sig { returns(String) }
  def self.generate_code
    6.times.map { rand(10) }.join
  end

  sig { returns(T.nilable(Account)) }
  def account
    @account ||= T.let(Account.find_by(email:), T.nilable(Account))
  end
end
