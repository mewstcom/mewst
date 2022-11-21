# typed: strict
# frozen_string_literal: true

class EmailConfirmation < ApplicationRecord
  EXPIRES_IN = T.let(2.hours, ActiveSupport::Duration)

  belongs_to :account, optional: true

  enum :event, {sign_up: 0, sign_in: 1, update_email: 2}

  validates :back, format: {with: %r{\A/}, allow_blank: true}
  validates :email, email: true, presence: true
  validates :event, presence: true
  validates :expires_at, presence: true
  validates :token, presence: true

  sig { returns(T::Boolean) }
  def expired?
    expires_at.past?
  end
end
