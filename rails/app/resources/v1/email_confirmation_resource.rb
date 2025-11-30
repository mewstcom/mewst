# typed: strict
# frozen_string_literal: true

class V1::EmailConfirmationResource < V1::ApplicationResource
  delegate :id, :email, :event, to: :email_confirmation

  sig { params(email_confirmation: EmailConfirmationRecord).void }
  def initialize(email_confirmation:)
    @email_confirmation = email_confirmation
  end

  sig { returns(T.nilable(String)) }
  def succeeded_at
    email_confirmation.succeeded_at&.iso8601
  end

  sig { returns(EmailConfirmationRecord) }
  attr_reader :email_confirmation
  private :email_confirmation
end
