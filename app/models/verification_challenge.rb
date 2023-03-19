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

  private

  delegate :code, to: :verification!

  sig { void }
  def valid_challenged_code
    return if challenged_code == code

    errors.add(:base, :equal_to_code)
  end
end
