# typed: strict
# frozen_string_literal: true

class Account::Activation
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes

  sig { returns(T.nilable(PhoneNumberVerification)) }
  attr_accessor :phone_number_verification

  delegate :phone_number, to: :phone_number_verification!

  attribute :atname, :string
  attribute :locale, :string

  validates :phone_number_verification, presence: true

  sig { returns(Account) }
  def run
    validate!

    account = T.let(nil, T.nilable(Account))
    ActiveRecord::Base.transaction do
      account = Account.create!(phone_number:)

      profile = account.create_profile!(atname:, profilable_type: Profile::PROFILABLE_TYPE_ACCOUNT, locale:)
      account.create_account_profile!(profile:)

      phone_number_verification!.destroy
    end

    T.cast(account, Account)
  rescue ActiveRecord::RecordInvalid => e
    e.record.errors.full_messages.each do |full_message|
      errors.add(:base, full_message)
    end

    fail ActiveModel::ValidationError.new(self)
  end

  private

  sig { returns(PhoneNumberVerification) }
  def phone_number_verification!
    T.cast(phone_number_verification, PhoneNumberVerification)
  end
end
