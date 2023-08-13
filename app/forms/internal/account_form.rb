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

  sig { returns(String) }
  def atname!
    T.must(atname)
  end

  sig { returns(String) }
  def email!
    T.must(email)
  end

  sig { returns(String) }
  def locale!
    T.must(locale)
  end

  sig { returns(String) }
  def password!
    T.must(password)
  end

  sig { void }
  private def atname_uniqueness
    if Profile.find_by(atname:)
      errors.add(:atname, :atname_uniqueness)
    end
  end

  sig { void }
  private def email_uniqueness
    if User.find_by(email:)
      errors.add(:email, :email_uniqueness)
    end
  end
end
