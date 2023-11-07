# typed: strict
# frozen_string_literal: true

class EmailConfirmation < ApplicationRecord
  extend Enumerize

  EVENT_PASSWORD_RESET = :password_reset
  EVENT_SIGN_UP = :sign_up
  EXPIRES_IN = T.let(15.minutes, ActiveSupport::Duration)

  enumerize :event, in: [EVENT_PASSWORD_RESET, EVENT_SIGN_UP]

  scope :active, -> { where("created_at > ?", EXPIRES_IN.ago) }
  scope :succeeded, -> { where.not(succeeded_at: nil) }

  validates :email, presence: true
  validates :code, format: {with: /\A\d{6}\z/}, presence: true

  sig { returns(String) }
  def self.generate_code
    6.times.map { rand(10) }.join
  end

  sig { returns(T::Boolean) }
  def succeeded?
    !succeeded_at.nil?
  end

  sig { params(locale: String).void }
  def send_sign_up_email!(locale:)
    self.code = EmailConfirmation.generate_code
    self.event = EmailConfirmation::EVENT_SIGN_UP
    save!

    EmailConfirmationMailer.email_confirmation(email_confirmation_id: id.not_nil!, locale:).deliver_later

    nil
  end

  sig { returns(T::Boolean) }
  def success
    update!(succeeded_at: Time.current) unless succeeded?
    true
  end
end
