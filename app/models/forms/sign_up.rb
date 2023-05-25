# typed: strict
# frozen_string_literal: true

class Forms::SignUp < Forms::Base
  attribute :atname, :string
  attribute :email, :string
  attribute :locale, :string
  attribute :password, :string

  validates :atname, format: {with: Account::ATNAME_FORMAT}, length: {maximum: 30}, presence: true
  validates :email, email: true, presence: true
  validates :locale, presence: true
  validates :password, length: {in: 6..128}, presence: true
  validate :atname_uniqueness
  validate :email_uniqueness

  private

  def atname_uniqueness
    if Profile.find_by(atname:)
      errors.add(:atname, :atname_uniqueness)
    end
  end

  def email_uniqueness
    if Account.find_by(email:)
      errors.add(:email, :email_uniqueness)
    end
  end
end
