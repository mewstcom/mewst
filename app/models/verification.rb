# typed: strict
# frozen_string_literal: true

class Verification < ApplicationRecord
  extend Enumerize

  EXPIRES_IN = 15.minutes

  enumerize :event, in: %i[sign_up]

  scope :active, -> { where("created_at > ?", EXPIRES_IN.ago) }

  validates :email, presence: true
  validates :code, format: {with: /\A\d{6}\z/}, presence: true

  sig { returns(String) }
  def self.generate_code
    6.times.map { rand(10) }.join
  end

  sig { returns(T.nilable(Account)) }
  def account
    @account ||= T.let(Account.find_by(phone_number:), T.nilable(Account))
  end
end
