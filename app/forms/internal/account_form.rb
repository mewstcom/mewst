# typed: strict
# frozen_string_literal: true

class Internal::AccountForm < Internal::ApplicationForm
  attribute :atname, :string
  attribute :email, :string
  attribute :locale, :string
  attribute :password, :string

  validates :atname, format: {with: Profile::ATNAME_FORMAT}, length: {maximum: 30}, presence: true
  validates :email, email: true, presence: true
  validates :locale, presence: true
  validates :password, length: {in: 6..128}, presence: true
  validate :atname_uniqueness
  validate :email_uniqueness

  sig { void }
  private def atname_uniqueness
    if Profile.find_by(atname:)
      errors.add(:atname, :uniqueness)
    end
  end

  sig { void }
  private def email_uniqueness
    if User.find_by(email:)
      errors.add(:email, :uniqueness)
    end
  end
end
