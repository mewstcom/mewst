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
      current_time = Time.current

      account = Account.create!(email:, locale:, password:, signed_up_at: current_time)
      profile = account.profiles.new(atname:, joined_at: current_time)
      account.profile_members.create!(profile:, joined_at: current_time)

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
