# typed: strict
# frozen_string_literal: true

class EmailConfirmationRecord < ApplicationRecord
  self.table_name = "email_confirmations"

  extend Enumerize

  EXPIRES_IN = T.let(15.minutes, ActiveSupport::Duration)

  enumerize :event, in: EmailConfirmationEvent.values.map(&:serialize)

  scope :active, -> { where(succeeded_at: nil).where("created_at > ?", EXPIRES_IN.ago) }
  scope :succeeded, -> { where.not(succeeded_at: nil) }

  validates :email, presence: true
  validates :code, format: {with: /\A\d{6}\z/}, presence: true

  sig { returns(String) }
  def self.generate_code
    6.times.map { rand(10) }.join
  end

  sig { returns(EmailConfirmationEvent) }
  def deserialized_event
    EmailConfirmationEvent.deserialize(event)
  end

  sig { returns(T::Boolean) }
  def succeeded?
    !succeeded_at.nil?
  end

  sig { returns(T::Boolean) }
  def success!
    update!(succeeded_at: Time.current) unless succeeded?
    true
  end

  sig { params(viewer: T.nilable(Actor)).returns(T::Boolean) }
  def process_after_success!(viewer:)
    if viewer && deserialized_event == EmailConfirmationEvent::EmailUpdate
      viewer.update_email!(email:)
    end

    true
  end

  sig { params(locale: Locale).returns(T::Boolean) }
  def send_mail!(locale:)
    EmailConfirmationMailer.email_confirmation(email_confirmation_id: id.not_nil!, locale: locale.serialize).deliver_later
    true
  end
end