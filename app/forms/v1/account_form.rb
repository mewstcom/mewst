# typed: strict
# frozen_string_literal: true

class V1::AccountForm < V1::ApplicationForm
  include FormConcerns::PasswordValidatable

  attribute :atname, :string
  attribute :email, :string
  attribute :locale, :string
  attribute :password, :string
  attribute :time_zone, :string

  validates :atname,
    format: {with: Profile::ATNAME_FORMAT},
    length: {in: Profile::ATNAME_MIN_LENGTH..Profile::ATNAME_MAX_LENGTH},
    presence: true,
    unreserved_atname: true
  validates :email, email: true, presence: true
  validates :locale, presence: true
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
