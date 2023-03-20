# typed: strict
# frozen_string_literal: true

class AccountActivation
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes

  sig { returns(T.nilable(Verification)) }
  attr_accessor :verification

  delegate :email, to: :verification!

  attribute :atname, :string
  attribute :locale, :string
  attribute :password, :string

  validates :atname, presence: true
  validates :password, length: {in: 6..128}, presence: true
  validates :verification, presence: true

  sig { returns(Account) }
  def run
    validate!

    account = T.let(nil, T.nilable(Account))
    ActiveRecord::Base.transaction do
      account = Account.create!(email:, locale:, password:)
      account.profiles.create!(atname:)

      verification!.destroy
    end

    T.cast(account, Account)
  rescue ActiveRecord::RecordInvalid => e
    e.record.errors.full_messages.each do |full_message|
      errors.add(:base, full_message)
    end

    fail ActiveModel::ValidationError.new(self)
  end

  private

  sig { returns(Verification) }
  def verification!
    T.cast(verification, Verification)
  end
end
