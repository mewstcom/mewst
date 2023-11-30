# typed: strict
# frozen_string_literal: true

class Internal::AccountForm < Internal::ApplicationForm
  attribute :atname, :string
  attribute :email, :string
  attribute :locale, :string
  attribute :password, :string
  attribute :time_zone, :string

  validates :atname,
    format: {with: Profile::ATNAME_FORMAT},
    length: {in: Profile::MIN_ATNAME_LENGTH..Profile::MAX_ATNAME_LENGTH},
    presence: true
  validates :email, email: true, presence: true
  validates :locale, presence: true
  validates :password,
    length: {in: 8..ActiveModel::SecurePassword::MAX_PASSWORD_LENGTH_ALLOWED},
    presence: true
  validates :time_zone, presence: true
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
