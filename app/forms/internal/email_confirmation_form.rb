# typed: strict
# frozen_string_literal: true

class Internal::EmailConfirmationForm < Internal::ApplicationForm
  attribute :email, :string
  attribute :event, :string
  attribute :locale, :string

  validates :email, email: true, presence: true
  validates :event, inclusion: {in: EmailConfirmation.event.values}, presence: true
  validates :locale, inclusion: {in: Locale.values.map(&:serialize)}, presence: true
end
