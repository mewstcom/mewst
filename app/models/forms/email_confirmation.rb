# typed: strict
# frozen_string_literal: true

class Forms::EmailConfirmation < Forms::Base
  attribute :email, :string
  attribute :locale, :string

  validates :email, email: true, presence: true
  validates :locale, presence: true

  sig { returns(Locale) }
  def locale!
    Locale.deserialize(locale&.downcase)
  end
end
