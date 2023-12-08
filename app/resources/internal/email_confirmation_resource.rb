# typed: strict
# frozen_string_literal: true

class Internal::EmailConfirmationResource < Internal::ApplicationResource
  delegate :id, :email, :event, to: :email_confirmation

  sig { params(email_confirmation: EmailConfirmation).void }
  def initialize(email_confirmation:)
    @email_confirmation = email_confirmation
  end

  sig { returns(T.nilable(String)) }
  def succeeded_at
    email_confirmation.succeeded_at&.iso8601
  end

  sig { returns(EmailConfirmation) }
  attr_reader :email_confirmation
  private :email_confirmation
end
