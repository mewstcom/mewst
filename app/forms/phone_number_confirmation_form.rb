# typed: strict
# frozen_string_literal: true

class PhoneNumberConfirmationForm < ApplicationForm
  attribute :phone_number, :string
  attribute :phone_number_full, :string

  validates :phone_number, presence: true
  validates :phone_number_full, presence: true
end
