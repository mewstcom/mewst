# typed: strict
# frozen_string_literal: true

class Forms::EmailConfirmationChallenge < Forms::Base
  attribute :email_confirmation_id, :string
  attribute :confirmation_code, :string

  validate :valid_confirmation_code

  sig { returns(EmailConfirmation) }
  def email_confirmation!
    T.let(EmailConfirmation.find_by(id: email_confirmation_id), EmailConfirmation)
  end

  private

  sig { returns(T.nilable(EmailConfirmation)) }
  def active_email_confirmation
    @active_email_confirmation ||= EmailConfirmation.active.find_by(id: email_confirmation_id)
  end

  sig { void }
  def valid_confirmation_code
    return if confirmation_code == active_email_confirmation&.code

    errors.add(:confirmation_code, :incorrect_or_expired)
  end
end
