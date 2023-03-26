# typed: strict
# frozen_string_literal: true

class VerificationChallenge
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes

  sig { returns(T.nilable(Verification)) }
  attr_accessor :verification

  attribute :challenged_code, :string

  delegate :account, to: :verification!

  validate :valid_challenged_code

  sig { returns(Verification) }
  def verification!
    T.cast(verification, Verification)
  end

  def success
    verification!.update!(succeeded_at: Time.current)
  end

  private

  sig { void }
  def valid_challenged_code
    return if challenged_code == verification&.code

    errors.add(:challenged_code, :incorrect_or_expired)
  end
end
