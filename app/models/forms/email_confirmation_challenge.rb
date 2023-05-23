# typed: strict
# frozen_string_literal: true

class Forms::EmailConfirmationChallenge < Forms::Base
  attribute :email_confirmation_id, :string
  attribute :confirmation_code, :string

  validate :valid_confirmation_code

  sig { returns(Verification) }
  def email_confirmation!
    T.let(Verification.find_by(id: email_confirmation_id), Verification)
  end

  private

  sig { returns(T.nilable(Verification)) }
  def active_email_confirmation
    @active_email_confirmation ||= Verification.active.find_by(id: email_confirmation_id)
  end

  sig { void }
  def valid_confirmation_code
    return if confirmation_code == active_email_confirmation&.code

    errors.add(:confirmation_code, :incorrect_or_expired)
  end
end
