# typed: strict
# frozen_string_literal: true

class Forms::EmailConfirmation < Forms::Base
  attribute :email, :string
  attribute :locale, :string

  validates :email, email: true, presence: true
  validates :locale, inclusion: {in: Locale.values.map(&:serialize)}, presence: true
end
