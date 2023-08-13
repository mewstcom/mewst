# typed: strict
# frozen_string_literal: true

class Internal::EmailConfirmationForm < Internal::ApplicationForm
  attribute :email, :string
  attribute :locale, :string

  validates :email, email: true, presence: true
  validates :locale, inclusion: {in: Locale.values.map(&:serialize)}, presence: true

  sig { returns(String) }
  def email!
    T.must(email)
  end

  sig { returns(String) }
  def locale!
    T.must(locale)
  end
end
